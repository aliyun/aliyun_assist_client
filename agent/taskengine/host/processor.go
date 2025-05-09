package host

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"github.com/aliyun/aliyun_assist_client/thirdparty/sirupsen/logrus"
	"github.com/hectane/go-acl"

	"github.com/aliyun/aliyun_assist_client/agent/flagging"
	"github.com/aliyun/aliyun_assist_client/agent/log"
	"github.com/aliyun/aliyun_assist_client/agent/metrics"
	"github.com/aliyun/aliyun_assist_client/agent/pluginmanager/acspluginmanager"
	"github.com/aliyun/aliyun_assist_client/agent/taskengine/models"
	"github.com/aliyun/aliyun_assist_client/agent/taskengine/scriptmanager"
	"github.com/aliyun/aliyun_assist_client/agent/taskengine/taskerrors"
	"github.com/aliyun/aliyun_assist_client/agent/util/errnoutil"
	"github.com/aliyun/aliyun_assist_client/agent/util/powerutil"
	"github.com/aliyun/aliyun_assist_client/agent/util/process"
	"github.com/aliyun/aliyun_assist_client/common/executil"
	"github.com/aliyun/aliyun_assist_client/common/langutil"
	"github.com/aliyun/aliyun_assist_client/common/pathutil"
)

type HostProcessor struct {
	TaskId string
	// Fundamental properties of command process
	CommandType     string
	CommandContent  string
	InvokeVersion   int
	Repeat          models.RunTaskRepeatType
	Timeout         int
	TerminationMode string
	// Additional attributes for command process in host
	CommandName         string
	WorkingDirectory    string
	Username            string
	WindowsUserPassword string

	// Launcher params
	Launcher string

	// Detected properties for command process in host
	envHomeDir     string
	realWorkingDir string

	// Generated variables to invoke command process
	scriptFilePath    string
	invokeCommand     string
	invokeCommandArgs []string

	// Object for command process
	processCmd *process.ProcessCmd

	// Generated variables from invoked command process
	exitCode     int
	resultStatus int

	canceled bool
}

var (
	launcherParamRe    = regexp.MustCompile(`'[^']*'|"[^"]*"|{{.*}}|\S+`)
	scriptFileRe       = regexp.MustCompile(`(?i){{\s*ACS\s*::\s*ScriptFileName\s*(?:\|\s*ext\s*\(([\w-.]+)\)\s*)?\s*}}`)
	environmentParamRe = regexp.MustCompile(`\{\{(.+?)\}\}`)
	// scriptFileRe       = regexp.MustCompile(`{{\s*([\w-.]+)\s*::\s*([\w-.]+)\s*(?:\|\s*([\w-.]+)\s*\(\s*([\w-.]+)\s*\)\s*)*}}`)
	pluginRe = regexp.MustCompile(`@(\S+)`)
)

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
	if langutil.NeedTransformEncoding() {
		commandContent = langutil.UTF8ToLocal(commandContent)
	}
	p.CommandContent = commandContent

	var processCmd *process.ProcessCmd
	var processCmdErr error
	switch p.TerminationMode {
	case "ProcessTree":
		groupName := fmt.Sprintf("%s-%d", p.TaskId, time.Now().Unix())
		processCmd, processCmdErr = process.NewProcessCmdWithProcessTree(groupName)
	case "", "Process":
		processCmd = process.NewProcessCmd()
	default:
		processCmdErr = fmt.Errorf("unknown terminate mode %s", p.TerminationMode)
	}
	taskLogger.Info("Terminate mode: ", p.TerminationMode)
	if processCmdErr != nil {
		return taskerrors.NewCreateProcessCollectionError(processCmdErr)
	}
	p.processCmd = processCmd

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
	var launcherParams []string
	// If launcher exists, scriptFileExtension is different
	environmentParamExist := false
	if p.Launcher != "" {
		// If exists {{ACS::ScriptFileName}}, while without Ext
		extensionMatches := scriptFileRe.FindStringSubmatch(p.Launcher)
		if len(extensionMatches) == 2 {
			scriptFileExtension = extensionMatches[1]
		}
		fileNameMatches := scriptFileRe.FindAllString(p.Launcher, -1)

		environmentMatches := environmentParamRe.FindAllString(p.Launcher, -1)
		// Launcher has other environment parameter and it's not {{ACS::ScriptFileName}} or {{ACS::ScriptFileName|Ext}}
		if len(fileNameMatches) == 1 {
			environmentParamExist = true
		}
		if len(fileNameMatches) > 1 || len(environmentMatches) != len(fileNameMatches) {
			taskLogger.Errorf("Invalid match when resolving environment parameter in launcher %s", p.Launcher)
			return taskerrors.NewInvalidEnvironmentParameterError(fmt.Sprintf(`Invalid match when resolving environment parameter in launcher %s`, p.Launcher))
		}
	}
	if scriptDir == "" {
		var err error
		scriptDir, err = pathutil.GetScriptPath()
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
		iVersion := ""
		commandName := ""
		if p.InvokeVersion > 0 {
			iVersion = fmt.Sprintf(".iv%d", p.InvokeVersion)
		}
		if len(p.CommandName) > 0 {
			commandName = p.CommandName + "-"
		}
		p.scriptFilePath = filepath.Join(scriptDir, fmt.Sprintf("%s%s%s%s", commandName, p.TaskId, iVersion, scriptFileExtension))

		// In english environment PowerShell scripts need to be saved in utf8-bom format,
		// otherwise no-ASCII characters will not be recognized
		commandContent = p.CommandContent
		if p.CommandType == "RunPowerShellScript" && !langutil.NeedTransformEncoding() {
			commandContent = string(append([]byte{0xEF, 0xBB, 0xBF}, []byte(commandContent)...))
		}
		if err := scriptmanager.SaveScriptFile(p.scriptFilePath, commandContent); err != nil {
			// NOTE: Only non-repeated tasks need to check whether command script
			// file exists.
			taskLogger.Error("Save script file error: ", err)
			if errors.Is(err, scriptmanager.ErrScriptFileExists) {
				if p.Repeat == models.RunTaskOnce || p.Repeat == models.RunTaskNextRebootOnly {
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
	if p.Launcher != "" {
		if !useScriptFile {
			metrics.GetTaskFailedEvent(
				"taskid", p.TaskId,
				"errormsg", fmt.Sprintf("Can not use script file, so Launcher script cannot run"),
				"reason", whyNoScriptFile.Error(),
			).ReportEvent()
			return whyNoScriptFile
		}
		launcher := scriptFileRe.ReplaceAllString(p.Launcher, p.scriptFilePath)
		launcherParams = launcherParamRe.FindAllString(launcher, -1)
		if !environmentParamExist {
			launcherParams = append(launcherParams, p.scriptFilePath)
		}
	}
	taskLogger.Infof("Launcher params: %v", launcherParams)
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

		if _, err := executil.LookPath(p.invokeCommand); err != nil {
			return taskerrors.NewSystemDefaultShellNotFoundError(err)
		}
	} else if p.CommandType == "RunPowerShellScript" {
		p.invokeCommand = "powershell"
		if useScriptFile {
			p.invokeCommandArgs = []string{"-file", p.scriptFilePath}
		} else {
			p.invokeCommandArgs = []string{"-command", p.CommandContent}
		}

		if _, err := executil.LookPath(p.invokeCommand); err != nil {
			return taskerrors.NewPowershellNotFoundError(err)
		}
		tempProcessCmd := process.NewProcessCmd()
		if err := tempProcessCmd.SyncRunSimple("powershell.exe", []string{"Set-ExecutionPolicy", "RemoteSigned"}, 10); err != nil {
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

	// For Launcher: change invokeCommand or invokeCommandArgs
	if p.Launcher != "" && len(launcherParams) > 0 {
		var cmdPath string
		// Check if it's plugin or local launcher
		if launcherParams[0][0] == '@' && len(launcherParams[0]) > 1 && launcherParams[0][1] != '@' {
			//It's a plugin and install it
			pluginName := matchPlugin(p.Launcher)

			pluginManager, err := acspluginmanager.NewPluginManager(false)
			if err != nil {
				return taskerrors.NewPluginLoadFailedError(fmt.Errorf("Broken plugin management: %w", err))
			}

			pluginInfo, err := acspluginmanager.QueryPluginFromLocal(pluginName, "")
			if pluginInfo == nil || err != nil {
				err1 := err
				pluginInfo, err = acspluginmanager.QueryPluginFromOnline(pluginName, "", "")
				// Online query nil
				if pluginInfo == nil || err != nil {
					taskLogger.Error("Plugin not found online and local: ", launcherParams[0])
					return taskerrors.NewPluginLoadFailedError(fmt.Errorf("query from local failed,%s; query from online failed,%s", err1, err))
				}
				err = pluginManager.InstallPluginFromOnline(pluginInfo, 20)
				if err != nil {
					taskLogger.Error("Plugin install error: ", err)
					return taskerrors.NewPluginLoadFailedError(fmt.Errorf("Plugin install error: %s", err))
				}
			}
			cmdPath = pluginManager.GetPluginCommandPath(pluginInfo)
			if _, err := executil.LookPath(cmdPath); err != nil {
				taskLogger.Error("LookPath error: ", err)
				return taskerrors.NewPluginLoadFailedError(err)
			}
		} else {
			if len(launcherParams[0]) > 1 && launcherParams[0][0] == '@' && launcherParams[0][1] == '@' {
				// local launcher start with @
				cmdPath = launcherParams[0][1:]
			} else {
				cmdPath = launcherParams[0]
			}
			if _, err := executil.LookPath(cmdPath); err != nil {
				taskLogger.Error("LookPath error: ", err)
				return taskerrors.NewLauncherNotFoundError(err)
			}
		}

		if useScriptFile {
			p.invokeCommand = cmdPath
			p.invokeCommandArgs = launcherParams[1:]
			taskLogger.Infof("InvokeCommand: %s ; InvokeCommandArgs: %s", p.invokeCommand, p.invokeCommandArgs)
		} else {
			metrics.GetTaskFailedEvent(
				"taskid", p.TaskId,
				"errormsg", fmt.Sprintf("Can not use script file, so Launcher script cannot run"),
				"reason", whyNoScriptFile.Error(),
			).ReportEvent()
			return whyNoScriptFile

		}

	}
	return nil
}

func matchPlugin(launcher string) string {

	match := pluginRe.FindStringSubmatch(launcher)
	if len(match) > 1 {
		result := match[1]
		return result
	} else {
		return ""
	}
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

func (p *HostProcessor) Cancel() error {
	if p.canceled {
		return nil
	}
	p.canceled = true
	// For periodic task, script is deleted when task canceled
	if p.isPeriodic() && !flagging.GetTaskKeepScriptFile() {
		os.Remove(p.scriptFilePath)
	}
	// For periodic task, p.Cancel() may be called before task.Run().
	if p.processCmd == nil {
		return nil
	}
	if err := p.processCmd.Cancel(); err != nil {
		return err
	}
	p.processCmd = nil
	return nil
}

func (p *HostProcessor) Cleanup(removeScriptFile bool) error {
	// For no-periodic task, script is deleted when task finished
	if !p.isPeriodic() && removeScriptFile {
		if err := os.Remove(p.scriptFilePath); err != nil {
			return err
		}
	}
	p.processCmd = nil
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

func (p *HostProcessor) isPeriodic() bool {
	return (p.Repeat == models.RunTaskCron || p.Repeat == models.RunTaskRate || p.Repeat == models.RunTaskAt)
}
