package commander

import (
	"context"
	"fmt"
	"io"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"

	"github.com/aliyun/aliyun_assist_client/agent/commandermanager"
	"github.com/aliyun/aliyun_assist_client/agent/log"
	"github.com/aliyun/aliyun_assist_client/agent/taskengine/models"
	"github.com/aliyun/aliyun_assist_client/agent/util/process"
	"github.com/aliyun/aliyun_assist_client/thirdparty/sirupsen/logrus"

	"github.com/aliyun/aliyun_assist_client/agent/taskengine/taskerrors"
)

var (
	// submissionId => processor
	processorMg_ sync.Map
)

type CommanderProcessor struct {
	TaskId string
	// Fundamental properties of command process
	CommandType      string
	CommandContent   string
	InvokeVersion    int
	Repeat           models.RunTaskRepeatType
	Timeout          int
	WorkingDirectory string
	Username         string
	Password         string

	// commander-specific parameters
	Annotation map[string]string

	commanderName    string
	commander        *commandermanager.Commander
	submissionId     string
	extraLubanParams string

	logger logrus.FieldLogger

	done        context.Context
	doneFunc    context.CancelFunc
	isResSetted atomic.Bool

	stdoutWriter io.Writer
	stderrWriter io.Writer

	exitCode  int
	resStatus int
	resError  *taskerrors.CommanderError
}

func NewCommanderProcessor(taskInfo models.RunTaskInfo, timeout int, annotation map[string]string, commanderName string) (*CommanderProcessor, error) {
	p := &CommanderProcessor{
		TaskId:        taskInfo.TaskId,
		InvokeVersion: taskInfo.InvokeVersion,
		CommandType:   taskInfo.CommandType,
		Repeat:        taskInfo.Repeat,
		Timeout:       timeout,

		WorkingDirectory: taskInfo.WorkingDir,
		Username:         taskInfo.Username,
		Password:         taskInfo.Password,

		Annotation:       annotation,
		commanderName:    commanderName,
		extraLubanParams: fmt.Sprintf("&commanderName=%s", commanderName),
	}

	id := uuid.NewString()[:8]
	p.submissionId = p.TaskId + "_" + id
	p.logger = log.GetLogger().WithFields(logrus.Fields{
		"submissionId": p.submissionId,
	})
	p.logger.Info(taskInfo.TaskId)

	return p, nil
}

func (p *CommanderProcessor) PreCheck() (string, error) {
	if commander, err := commandermanager.GetCommanderRunning(p.commanderName); err != nil {
		return "", err
	} else {
		p.commander = commander
	}
	client, err := p.commander.Client()
	if err != nil {
		return err.Code, err
	}
	extraLubanParams, err := client.PreCheckScript(p.logger, context.Background(),
		p.submissionId, p.CommandType, p.Timeout, p.WorkingDirectory, p.Username, p.Password, p.Annotation)
	if err != nil {
		p.logger.WithError(err).Error("precheck script failed")
		return err.Code, err
	}
	p.extraLubanParams += fmt.Sprintf("&commanderVersion=%s", p.commander.Version())
	p.extraLubanParams += extraLubanParams

	return "", nil
}

func (p *CommanderProcessor) Prepare(content string) error {
	client, err := p.commander.Client()
	if err != nil {
		return err
	}
	err = client.PrepareScript(p.logger, context.Background(), p.submissionId, content)
	if err != nil {
		p.logger.WithError(err).Error("prepare script failed")
		return err
	}
	p.CommandContent = content

	return nil
}

func (p *CommanderProcessor) SyncRun(
	stdoutWriter io.Writer,
	stderrWriter io.Writer,
	stdinReader io.Reader) (exitCode int, resStatus int, resError error) {

	p.stdoutWriter = stdoutWriter
	p.stderrWriter = stderrWriter
	p.done, p.doneFunc = context.WithCancel(context.Background())
	defer p.doneFunc()

	storeTask(p.submissionId, p)
	defer deleteTask(p.submissionId)

	if err := p.submitScript(); err != nil {
		p.logger.WithError(err).Error("submit script failed")
		// the exitCode is only valid when process.Success
		return -1, process.Fail, err
	}

	defer func() {
		exitCode = p.exitCode
		resStatus = p.resStatus
		resError = p.resError
	}()

	p.commander.RegisterExitHandler(p.submissionId, func() {
		p.SetFinalStatus(-1, process.Fail, taskerrors.NewCommanderExitError("commander exit before task finished"))
	})
	defer p.commander.DeRegisterExitHandler(p.submissionId)
	timeout := time.NewTimer(time.Duration(p.Timeout+3) * time.Second)
	for {
		select {
		case <-timeout.C:
			if p.isResSetted.CompareAndSwap(false, true) {
				p.resStatus = process.Timeout
				p.resError = taskerrors.NewStatusUpdateTimeoutError("commander not report task status before timeout")
			}
			return
		case <-p.done.Done():
			return
		}
	}
}

func (p *CommanderProcessor) Cancel() error {
	client, err := p.commander.Client()
	if err != nil {
		return err
	}

	err = client.CancelSubmitted(p.logger, context.Background(), p.submissionId)
	if err != nil {
		p.logger.WithError(err).Error("cancel task failed")
		return err
	}
	return nil
}

func (p *CommanderProcessor) Cleanup(bool) error {
	client, err := p.commander.Client()
	if err != nil {
		return err
	}

	err = client.CleanUpSubmitted(p.logger, context.Background(), p.submissionId)
	if err != nil {
		p.logger.WithError(err).Error("clean up failed")
		return err
	}
	return nil
}

func (p *CommanderProcessor) ExtraLubanParams() string {
	return p.extraLubanParams
}

func (p *CommanderProcessor) SideEffect() error {
	client, err := p.commander.Client()
	if err != nil {
		return err
	}

	err = client.DisposeSubmission(p.logger, context.Background(), p.submissionId)
	if err != nil {
		p.logger.WithError(err).Error("dispose submission failed")
		return err
	}
	return nil
}

func (p *CommanderProcessor) SetFinalStatus(exitCode, taskStatus int, err *taskerrors.CommanderError) {
	if p.isResSetted.CompareAndSwap(false, true) {
		p.exitCode = exitCode
		p.resStatus = taskStatus
		p.resError = err
		if p.doneFunc != nil {
			p.doneFunc()
		}
	}
}

func (p *CommanderProcessor) WriteOutput(index int, output string) {
	if p.stdoutWriter != nil {
		p.stdoutWriter.Write([]byte(output))
	} else if p.stderrWriter != nil {
		p.stderrWriter.Write([]byte(output))
	}
}

func (p *CommanderProcessor) submitScript() error {
	client, err := p.commander.Client()
	if err != nil {
		return err
	}

	err = client.SubmitScript(p.logger, context.Background(), p.submissionId)
	if err != nil {
		p.logger.WithError(err).Error("submit script failed")
		return err
	}
	return nil
}

func FindCommanderTask(submissionId string) (*CommanderProcessor, error) {
	if value, ok := processorMg_.Load(submissionId); !ok {
		return nil, fmt.Errorf("submission not exist")
	} else if task, ok := value.(*CommanderProcessor); !ok {
		return nil, fmt.Errorf("type convert failed")
	} else {
		return task, nil
	}
}

func storeTask(submissionId string, p *CommanderProcessor) error {
	if _, ok := processorMg_.LoadOrStore(submissionId, p); ok {
		return fmt.Errorf("submission duplicated")
	}
	return nil
}

func deleteTask(submission string) {
	processorMg_.Delete(submission)
}
