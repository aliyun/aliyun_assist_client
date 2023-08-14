package host

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/aliyun/aliyun_assist_client/thirdparty/sirupsen/logrus"
	"github.com/hectane/go-acl"

	"github.com/aliyun/aliyun_assist_client/agent/log"
	"github.com/aliyun/aliyun_assist_client/agent/metrics"
	"github.com/aliyun/aliyun_assist_client/agent/taskengine/models"
	"github.com/aliyun/aliyun_assist_client/agent/taskengine/scriptmanager"
	"github.com/aliyun/aliyun_assist_client/agent/taskengine/taskerrors"
	"github.com/aliyun/aliyun_assist_client/agent/util"
	"github.com/aliyun/aliyun_assist_client/agent/util/errnoutil"
	"github.com/aliyun/aliyun_assist_client/agent/util/powerutil"
	"github.com/aliyun/aliyun_assist_client/agent/util/process"
)

type HostProcessor struct {
	TaskId string
	// Fundamental properties of command process
	CommandType    string
	CommandContent string
	Repeat         models.RunTaskRepeatType
	Timeout        int
	// Additional attributes for command process in host
	CommandName         string
	WorkingDirectory    string
	Username            string
	WindowsUserPassword string

	// Detected properties for command process in host
	envHomeDir     string
	realWorkingDir string

	// Generated variables to invoke command process
	scriptFilePath    string
	invokeCommand     string
	invokeCommandArgs []string

	// Object for command process
	processCmd process.ProcessCmd

	// Generated variables from invoked command process
	exitCode     int
	resultStatus int
}

func (p *HostProcessor) PreCheck() (string, error) {
	taskLogger := log.GetLogger().WithFields(logrus.Fields{
		"TaskId": p.TaskId,
		"Phase":  "HostProcessor-PreCheck",
	})

	if len(p.Username) > 0 {
		if _, err := p.checkCredentials(); err != nil {
			if validationErr, ok := err.(taskerrors.NormalizedValidationError); ok {
				return validationErr.Param(), err
			} else {
				return "UsernameOrPasswordInvalid", err
			}
		}
	}

	var err error
	p.envHomeDir, err = p.checkHomeDirectory()
	if err != nil {
		taskLogger.WithError(err).Warningln("Invalid HOME directory for invocation")
	}

	p.realWorkingDir, err = p.checkWorkingDirectory()
	if err != nil {
		return "workingDirectory", err
	}

	return "", nil
}

func (p *HostProcessor) Prepare(commandContent string) error {
	taskLogger := log.GetLogger().WithFields(logrus.Fields{
		"TaskId": p.TaskId,
		"Phase":  "HostProcessor-Preparing",
	})
	p.CommandContent = commandContent

	useScriptFile := true
	var whyNoScriptFile error
	defer func() {
		if !useScriptFile {
			metrics.GetTaskWarnEvent(
				"taskid", p.TaskId,
				"warnmsg", "NotUseScriptFile",
				"reason", whyNoScriptFile.Error(),
			).ReportEvent()
		}
	}()

	var scriptDir string
	var scriptFileExtension string
	switch p.CommandType {
	case "RunBatScript":
		scriptFileExtension = ".bat"
	case "RunPowerShellScript":
		scriptFileExtension = ".ps1"
	case "RunShellScript":
		scriptFileExtension = ".sh"

		if p.Username != "" {
			scriptDir = "/tmp"
		}
	default:
		return taskerrors.NewUnknownCommandTypeError()
	}

	if scriptDir == "" {
		var err error
		scriptDir, err = util.GetScriptPath()
		if err != nil {
			useScriptFile = false
			if errnoutil.IsNoEnoughSpaceError(err) {
				whyNoScriptFile = taskerrors.NewNoEnoughSpaceError(err)
			} else {
				whyNoScriptFile = taskerrors.NewGetScriptPathError(err)
			}
			taskLogger.Error("Get script dir path error: ", whyNoScriptFile)
		}
	}

	if useScriptFile {
		if p.CommandName == "" {
			p.scriptFilePath = filepath.Join(scriptDir, p.TaskId+scriptFileExtension)
		} else {
			p.scriptFilePath = filepath.Join(scriptDir, fmt.Sprintf("%s-%s%s", p.CommandName, p.TaskId, scriptFileExtension))
		}

		if err := scriptmanager.SaveScriptFile(p.scriptFilePath, p.CommandContent); err != nil {
			// NOTE: Only non-repeated tasks need to check whether command script
			// file exists.
			taskLogger.Error("Save script file error: ", err)
			if errors.Is(err, scriptmanager.ErrScriptFileExists) {
				if (p.Repeat == models.RunTaskOnce || p.Repeat == models.RunTaskNextRebootOnly) {
					return taskerrors.NewScriptFileExistedError(p.scriptFilePath, err)
				}
			} else if errnoutil.IsNoEnoughSpaceError(err) {
				useScriptFile = false
				whyNoScriptFile = taskerrors.NewNoEnoughSpaceError(err)
			} else {
				useScriptFile = false
				whyNoScriptFile = taskerrors.NewSaveScriptFileError(err)
			}
			// If save script into file failed, file may be created but is invalid
			if !useScriptFile {
				os.Remove(p.scriptFilePath)
			}
		}
	}

	if useScriptFile {
		if p.CommandType == "RunShellScript" {
			if err := acl.Chmod(p.scriptFilePath, 0755); err != nil {
				useScriptFile = false
				whyNoScriptFile = taskerrors.NewSetExecutablePermissionError(err)
			}
		} else {
			if p.Username != "" {
				if err := acl.Chmod(p.scriptFilePath, 0755); err != nil {
					useScriptFile = false
					whyNoScriptFile = taskerrors.NewSetWindowsPermissionError(err)
				}
			}
		}
	}

	p.invokeCommand = p.scriptFilePath
	p.invokeCommandArgs = []string{}
	if p.CommandType == "RunShellScript" {
		p.invokeCommand = "sh"
		if useScriptFile {
			p.invokeCommandArgs = []string{"-c", p.scriptFilePath}
		} else {
			p.invokeCommandArgs = []string{"-c", p.CommandContent}
		}

		if _, err := exec.LookPath(p.invokeCommand); err != nil {
			return taskerrors.NewSystemDefaultShellNotFoundError(err)
		}
	} else if p.CommandType == "RunPowerShellScript" {
		p.invokeCommand = "powershell"
		if useScriptFile {
			p.invokeCommandArgs = []string{"-file", p.scriptFilePath}
		} else {
			p.invokeCommandArgs = []string{"-command", p.CommandContent}
		}

		if _, err := exec.LookPath(p.invokeCommand); err != nil {
			return taskerrors.NewPowershellNotFoundError(err)
		}

		if err := p.processCmd.SyncRunSimple("powershell.exe", []string{"Set-ExecutionPolicy", "RemoteSigned"}, 10); err != nil {
			taskLogger.WithError(err).Warningln("Failed to set powershell execution policy")
		}
	} else if !useScriptFile {
		metrics.GetTaskFailedEvent(
			"taskid", p.TaskId,
			"errormsg", fmt.Sprintf("Can not use script file, so this type[%s] script cannot run", p.CommandType),
			"reason", whyNoScriptFile.Error(),
		).ReportEvent()
		return whyNoScriptFile
	}

	return nil
}

func (p *HostProcessor) SyncRun(
	stdoutWriter io.Writer,
	stderrWriter io.Writer,
	stdinReader io.Reader) (int, int, error) {
	if p.Username != "" {
		p.processCmd.SetUserInfo(p.Username)
	}
	if p.WindowsUserPassword != "" {
		p.processCmd.SetPasswordInfo(p.WindowsUserPassword)
	}
	// Fix $HOME environment variable undex *nix
	if p.envHomeDir != "" {
		p.processCmd.SetHomeDir(p.envHomeDir)
	}

	var err error
	p.exitCode, p.resultStatus, err = p.processCmd.SyncRun(p.realWorkingDir, p.invokeCommand, p.invokeCommandArgs, stdoutWriter, stderrWriter, stdinReader, nil, p.Timeout)
	if p.resultStatus == process.Fail && err != nil {
		err = taskerrors.NewExecuteScriptError(err)
	}

	return p.exitCode, p.resultStatus, err
}

func (p *HostProcessor) Cancel() {
	p.processCmd.Cancel()
}

func (p *HostProcessor) Cleanup(removeScriptFile bool) error {
	if removeScriptFile {
		if err := os.Remove(p.scriptFilePath); err != nil {
			return err
		}
	}

	return nil
}

func (p *HostProcessor) SideEffect() error {
	taskLogger := log.GetLogger().WithFields(logrus.Fields{
		"TaskId": p.TaskId,
		"Phase":  "HostProcessor-SideEffect",
	})

	// Perform instructed poweroff/reboot action after task finished
	if p.resultStatus == process.Success {
		if p.exitCode == exitcodePoweroff {
			taskLogger.Infof("Poweroff the instance due to the special task exitcode %d", p.exitCode)
			powerutil.Shutdown(false)
		} else if p.exitCode == exitcodeReboot {
			taskLogger.Infof("Reboot the instance due to the special task exitcode %d", p.exitCode)
			powerutil.Shutdown(true)
		}
	}

	return nil
}

func (p *HostProcessor) ExtraLubanParams() string {
	return ""
}
