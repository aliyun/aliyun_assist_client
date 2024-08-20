package process

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/aliyun/aliyun_assist_client/agent/log"
	"github.com/aliyun/aliyun_assist_client/agent/util/process/processcollection"
	"github.com/aliyun/aliyun_assist_client/common/executil"
	"github.com/aliyun/aliyun_assist_client/thirdparty/sirupsen/logrus"
)

const (
	Success int = iota
	Fail
	Timeout
	groupsIdentifier = "groups="
)

type WaitProcessResult struct {
	processState *os.ProcessState
	err          error
}

type CmdOption func(*exec.Cmd) error

type ProcessCmd struct {
	canceledChan chan bool
	command      *exec.Cmd
	user_name    string
	password     string
	homeDir      string
	env          []string

	commandOptions []CmdOption
	collection     processcollection.ProcessCollection
}

func NewProcessCmd(options ...CmdOption) *ProcessCmd {
	p := &ProcessCmd{
		commandOptions: options,
	}
	return p
}

func NewProcessCmdWithProcessTree(groupName string, options ...CmdOption) (*ProcessCmd, error) {
	p := &ProcessCmd{
		commandOptions: options,
	}
	p.collection, _ = processcollection.CreateProcessTree(groupName)
	return p, nil
}

type cancellableAndSafeWriter struct {
	baseWriter    io.Writer
	cancelChannel chan bool
	cancelled     bool
	lock          sync.Mutex
}

type ReadCallbackFunc func(stdoutWriter io.Reader, stderrWriter io.Reader)

func (p *ProcessCmd) Cancel() error {
	if p.collection != nil {
		return p.collection.KillAll()
	} else if p.command != nil {
		if err := p.command.Process.Kill(); err != nil {
			return fmt.Errorf("Failed to kill process %d: %s", p.command.Process.Pid, err)
		}
	}
	return nil
}

func (p *ProcessCmd) SetUserInfo(name string) {
	p.user_name = name
}

func (p *ProcessCmd) SetPasswordInfo(password string) {
	p.password = password
}

func (p *ProcessCmd) SetHomeDir(homeDir string) {
	p.homeDir = homeDir
}

func (p *ProcessCmd) SetEnv(env []string) {
	p.env = env
}

func (p *ProcessCmd) SyncRunSimple(commandName string, commandArguments []string, timeOut int) error {
	p.command = executil.Command(commandName, commandArguments...)
	logger := log.GetLogger().WithFields(logrus.Fields{
		"command": p.command.Args,
		"timeout": timeOut,
	})

	if err := p.prepareProcess(); err != nil {
		return err
	}

	if err := p.command.Start(); err != nil {
		logger.WithError(err).Errorln("error occurred starting the command")
		return errors.New("error occurred starting the command")
	}

	if p.collection != nil {
		if err := p.collection.AddProcess(p.command.Process); err != nil {
			logger.WithError(err).Error("add process into collection failed")
			p.command.Process.Kill()
			return fmt.Errorf("add process into collection failed: %v", err)
		}
		logger.Infof("add process %d into collection", p.command.Process.Pid)
	}

	finished := make(chan error, 1)
	go func() {
		finished <- p.command.Wait()
	}()

	var err error
	select {
	case err = <-finished:
		logger.Infoln("Process completed.")
		if err != nil {
			logger.WithError(err).Infoln("error in run command")
		}
	case <-time.After(time.Duration(timeOut) * time.Second):
		logger.Errorln("Timeout in run command.")
		err = errors.New("cmd run timeout")
		if p.collection != nil {
			if err := p.collection.KillAll(); err != nil {
				logger.WithError(err).Error("kill all process in collection failed")
			}
		} else {
			p.command.Process.Kill()
		}
	}

	if p.collection != nil {
		p.collection.Dispose()
	}
	return err
}

func (p *ProcessCmd) SyncRun(
	workingDir string,
	commandName string,
	commandArguments []string,
	stdoutWriter io.Writer,
	stderrWriter io.Writer,
	stdinReader io.Reader,
	callbackFunc ReadCallbackFunc,
	timeOut int) (exitCode int, status int, err error) {

	status = Success
	exitCode = 0

	p.command = executil.Command(commandName, commandArguments...)
	p.command.Stdout = stdoutWriter
	p.command.Stderr = stderrWriter
	p.command.Stdin = stdinReader
	p.command.Dir = workingDir
	p.command.Env = p.env

	if err := p.prepareProcess(); err != nil {
		return 0, Fail, err
	}
	if p.user_name != "" {
		if err := p.addCredential(); err != nil {
			return 0, Fail, err
		}
	}

	if err = p.command.Start(); err != nil {
		log.GetLogger().Errorln("error occurred starting the command", err)
		exitCode = 1
		return exitCode, Fail, err
	}
	if p.collection != nil {
		if err := p.collection.AddProcess(p.command.Process); err != nil {
			log.GetLogger().WithError(err).Error("add process into collection failed")
			p.command.Process.Kill()
			return 1, Fail, fmt.Errorf("add process into collection failed: %v", err)
		} else {
			log.GetLogger().Infof("add process %d into collection %s success", p.command.Process.Pid, p.collection.Name())
		}
	}

	finished := make(chan WaitProcessResult, 1)
	go func() {
		processState, err := p.command.Process.Wait()
		finished <- WaitProcessResult{
			processState: processState,
			err:          err,
		}
	}()

	var timeoutChannel <-chan time.Time = nil
	if timeOut > 0 {
		timer := time.NewTimer(time.Duration(timeOut) * time.Second)
		defer timer.Stop()
		timeoutChannel = timer.C
	}
	select {
	case waitProcessResult := <-finished:
		log.GetLogger().Println("Command completed.", commandName)
		if waitProcessResult.processState != nil {
			if waitProcessResult.err != nil {
				log.GetLogger().WithFields(logrus.Fields{
					"processState": waitProcessResult.processState,
				}).WithError(waitProcessResult.err).Error("os.Process.Wait() returns error with valid process state")
			}

			exitCode = waitProcessResult.processState.ExitCode()
			// Sleep 200ms to allow remaining data to be copied back
			time.Sleep(time.Duration(200) * time.Millisecond)
			// Explicitly break select statement in case timer also times out
			break
		} else {
			exitCode = 1
			return exitCode, Fail, waitProcessResult.err
		}
	case <-timeoutChannel:
		log.GetLogger().Errorln("Timeout in run command.", commandName)
		exitCode = 1
		status = Timeout
		err = errors.New("timeout")
		if p.collection != nil {
			if err := p.collection.KillAll(); err != nil {
				log.GetLogger().WithError(err).Error("kill all process in collection failed")
			}
		} else {
			p.command.Process.Kill()
		}
	}

	if p.user_name != "" {
		p.removeCredential()
	}

	if p.collection != nil {
		p.collection.Dispose()
	}

	return exitCode, status, err
}

func (p *ProcessCmd) Pid() int {
	if p.command == nil || p.command.Process == nil {
		return -1
	}
	return p.command.Process.Pid
}

func GetUserCredentials(sessionUser string) (uint32, uint32, []uint32, error) {
	uidCmdArgs := append([]string{"-c"}, fmt.Sprintf("id -u %s", sessionUser))
	cmd := executil.Command("sh", uidCmdArgs...)
	out, err := cmd.Output()
	if err != nil {
		log.GetLogger().Errorf("Failed to retrieve uid for %s: %v", sessionUser, err)
		return 0, 0, nil, err
	}

	uid, err := strconv.Atoi(strings.TrimSpace(string(out)))
	if err != nil {
		log.GetLogger().Errorf("%s not found: %v", sessionUser, err)
		return 0, 0, nil, err
	}

	gidCmdArgs := append([]string{"-c"}, fmt.Sprintf("id -g %s", sessionUser))
	cmd = executil.Command("sh", gidCmdArgs...)
	out, err = cmd.Output()
	if err != nil {
		log.GetLogger().Errorf("Failed to retrieve gid for %s: %v", sessionUser, err)
		return 0, 0, nil, err
	}

	gid, err := strconv.Atoi(strings.TrimSpace(string(out)))
	if err != nil {
		log.GetLogger().Errorf("%s not found: %v", sessionUser, err)
		return 0, 0, nil, err
	}

	// Get the list of associated groups
	groupNamesCmdArgs := append([]string{"-c"}, fmt.Sprintf("id %s", sessionUser))
	cmd = executil.Command("sh", groupNamesCmdArgs...)
	out, err = cmd.Output()
	if err != nil {
		log.GetLogger().Errorf("Failed to retrieve groups for %s: %v", sessionUser, err)
		return 0, 0, nil, err
	}

	// Example format of output: uid=1873601143(ssm-user) gid=1873600513(domain users) groups=1873600513(domain users),1873601620(joiners),1873601125(aws delegated add workstations to domain users)
	// Extract groups from the output
	groupsIndex := strings.Index(string(out), groupsIdentifier)
	var groupIds []uint32

	if groupsIndex > 0 {
		// Extract groups names and ids from the output
		groupNamesAndIds := strings.Split(string(out)[groupsIndex+len(groupsIdentifier):], ",")

		// Extract group ids from the output
		for _, value := range groupNamesAndIds {
			groupId, err := strconv.Atoi(strings.TrimSpace(value[:strings.Index(value, "(")]))
			if err != nil {
				log.GetLogger().Errorf("Failed to retrieve group id from %s: %v", value, err)
				return 0, 0, nil, err
			}

			groupIds = append(groupIds, uint32(groupId))
		}
	}

	return uint32(uid), uint32(gid), groupIds, nil
}
