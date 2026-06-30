package shell

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"regexp"
	"runtime"
	"runtime/debug"
	"strings"
	"sync"
	"unsafe"

	sessionresult "github.com/aliyun/aliyun_assist_client/agent/session/sessionresult"
	"github.com/aliyun/aliyun_assist_client/agent/session/winpty"
	"github.com/aliyun/aliyun_assist_client/agent/util/process"
	"github.com/aliyun/aliyun_assist_client/agent/util/winapi"
	"github.com/aliyun/aliyun_assist_client/common/fileutil"
	"github.com/aliyun/aliyun_assist_client/common/pathutil"
	"github.com/aliyun/aliyun_assist_client/thirdparty/sirupsen/logrus"
	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

const (
	defaultConsoleCol = 200
	defaultConsoleRow = 60
)

type ShellPlugin struct {
	ShellPluginBase
	pty *winpty.WinPTY
}

func StartPty(plugin *ShellPlugin) *sessionresult.SessionResult {
	finalCmd := "powershell.exe"
	if plugin.cmdContent != "" {
		finalCmd = plugin.cmdContent
	}
	plugin.logger.Infoln("finalCmd ", finalCmd)
	exe_path, err := pathutil.GetExecutableDir()
	if err != nil {
		return sessionresult.NewWinptyLoadFailedError("winpty.dll", err)
	}

	if plugin.cmdContent == "" {
		_, err := exec.LookPath(finalCmd)
		if err != nil && !errors.Is(err, exec.ErrDot) {
			return sessionresult.NewShellCommandNotFoundError(finalCmd, err)
		}
	}

	winptyDllFilePath := filepath.Join(exe_path, "plugin", "SessionManager", "winpty.dll")
	winptyExeFilePath := filepath.Join(exe_path, "plugin", "SessionManager", "winpty-agent.exe")
	if !fileutil.CheckFileIsExist(winptyDllFilePath) {
		return sessionresult.NewWinptyLoadFailedError("winpty.dll", fmt.Errorf("%s not found", winptyDllFilePath))
	}
	if !fileutil.CheckFileIsExist(winptyExeFilePath) {
		return sessionresult.NewWinptyLoadFailedError("winpty-agent.exe", fmt.Errorf("%s not found", winptyExeFilePath))
	}
	var pty *winpty.WinPTY
	if plugin.username == "" {
		pty, err = winpty.Start(winptyDllFilePath, finalCmd, defaultConsoleCol, defaultConsoleRow, winpty.DEFAULT_WINPTY_FLAGS, nil, "")
		if err != nil {
			plugin.logger.Errorln("error in winpty.Start")
			return sessionresult.NewOpenPtyFailedError(err)
		}
	} else {
		targetAccount := plugin.username
		userSID, _, _, err := windows.LookupSID("", targetAccount)
		if err != nil {
			plugin.logger.WithError(err).Errorf("look up SID of user %s failed", targetAccount)
			return sessionresult.NewObtainUserIdentityFailedError(targetAccount, err)
		}

		var wg sync.WaitGroup
		var startPtyAsUserErr *sessionresult.SessionResult
		wg.Add(1)
		go func() {
			defer func() {
				if r := recover(); r != nil {
					plugin.logger.Errorf("Start pty as user panic: \n%v", r)
					plugin.logger.Errorf("Stacktrace:\n%s", debug.Stack())
				}
			}()
			defer wg.Done()
			pty, startPtyAsUserErr = plugin.startPtyAsUser(winptyDllFilePath, winptyExeFilePath, userSID, plugin.username, plugin.passwordName, finalCmd)
			if err != nil {
				plugin.logger.WithError(startPtyAsUserErr).Errorln("Start pty as user failed")
			}
		}()
		wg.Wait()
		if startPtyAsUserErr != nil {
			return startPtyAsUserErr
		}
	}

	plugin.pty = pty
	plugin.stdin = pty.StdIn
	plugin.stdout = pty.StdOut

	if plugin.first_ws_col != 0 {
		plugin.SetSize(plugin.first_ws_col, plugin.first_ws_row)
	}
	return nil
}

func (p *ShellPlugin) startPtyAsUser(winptyDllFilePath string, winptyExeFilePath string, userSID *windows.SID, user string, passwordName string, shellCmd string) (*winpty.WinPTY, *sessionresult.SessionResult) {
	pass, err := process.GetSecretParam(p.logger, passwordName)
	if err != nil {
		return nil, sessionresult.NewPasswordNameNotFoundError(passwordName, err)
	}
	// Check winpty-agent.exe permission
	if err = addReadExecuteRight(p.logger, userSID, winptyExeFilePath); err != nil {
		return nil, sessionresult.NewWinptyPermissionDeniedError(user, winptyExeFilePath, err)
	}
	if err = addReadExecuteRight(p.logger, userSID, winptyDllFilePath); err != nil {
		return nil, sessionresult.NewWinptyPermissionDeniedError(user, winptyDllFilePath, err)
	}

	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	// Impersonate current thread as runAs user
	p.logger.Infof("Impersonating %s", user)
	if err = winapi.Impersonate(p.logger, user, pass); err != nil {
		return nil, sessionresult.NewOpenPtyFailedError(fmt.Errorf("Failed to impersonate user %s, %v", user, err))
	}

	// Setup environment variables for the user context thread.
	env, pwd := setupSessionEnvs(p.logger, user)
	p.logger.Infof("Set %d envs for session, pwd is %s", len(env), pwd)
	// Start Winpty under the user context thread.
	var pty *winpty.WinPTY
	if pty, err = winpty.Start(winptyDllFilePath, shellCmd, defaultConsoleCol, defaultConsoleRow, winpty.WINPTY_FLAG_IMPERSONATE_THREAD, env, pwd); err != nil {
		p.logger.WithError(err).Error("Start winpty failed.")
		return nil, sessionresult.NewOpenPtyFailedError(err)
	}

	// Revert thread to original context
	if err = winapi.RevertToSelf(); err != nil {
		p.logger.WithError(err).Error("Reverting to system profile failed.")
		return nil, sessionresult.NewOpenPtyFailedError(fmt.Errorf("Failed to revert to self after starting winpty with impersonated user %s, %v", user, err))
	}
	p.logger.Info("Reverted to system profile.")
	return pty, nil
}

func (p *ShellPlugin) waitPid() {

}

func (p *ShellPlugin) stop() (err error) {
	p.logger.Info("Stopping winpty")
	if p.pty == nil {
		return nil
	}
	if err = p.pty.Close(); err != nil {
		return fmt.Errorf("Stop winpty failed: %s", err)
	}

	return nil
}

func (p *ShellPlugin) SetSize(ws_col, ws_row uint32) (err error) {
	if p.pty == nil {
		p.first_ws_col = ws_col
		p.first_ws_row = ws_row
		return nil
	}
	if err = p.pty.SetSize(ws_col, ws_row); err != nil {
		return fmt.Errorf("Set winpty size failed: %s", err)
	}
	return nil
}

func (p *ShellPlugin) onInputStreamData(payload []byte) error {
	// deal with powershell nextline issue https://github.com/lzybkr/PSReadLine/issues/579
	payloadString := string(payload)
	if strings.Contains(payloadString, "\r\n") {
		// From windows machine, do nothing
	} else if strings.Contains(payloadString, "\n") {
		// From linux machine, replace \n with \r
		num := strings.Index(payloadString, "\n")
		payloadString = strings.Replace(payloadString, "\n", "\r", num-1)
	}

	if _, err := p.stdin.Write([]byte(payloadString)); err != nil {
		p.logger.Errorf("Unable to write to stdin, err: %v.", err)
		return err
	}
	return nil
}

// Reference from go-winio
// https://github.com/microsoft/go-winio/blob/v0.6.2/internal/fs/fs.go#L58
const (
	STANDARD_RIGHTS_READ    windows.ACCESS_MASK = windows.READ_CONTROL
	STANDARD_RIGHTS_EXECUTE windows.ACCESS_MASK = windows.READ_CONTROL

	SYNCHRONIZE windows.ACCESS_MASK = 0x0010_0000

	FILE_READ_DATA windows.ACCESS_MASK = (0x0001) // file & pipe
	FILE_READ_EA   windows.ACCESS_MASK = (0x0008) // file & directory
	FILE_EXECUTE   windows.ACCESS_MASK = (0x0020) // file

	FILE_READ_ATTRIBUTES windows.ACCESS_MASK = (0x0080) // all

	FILE_GENERIC_READ    windows.ACCESS_MASK = (STANDARD_RIGHTS_READ | FILE_READ_DATA | FILE_READ_ATTRIBUTES | FILE_READ_EA | SYNCHRONIZE)
	FILE_GENERIC_EXECUTE windows.ACCESS_MASK = (STANDARD_RIGHTS_EXECUTE | FILE_READ_ATTRIBUTES | FILE_EXECUTE | SYNCHRONIZE)
)

func addReadExecuteRight(logger logrus.FieldLogger, targetSid *windows.SID, filePath string) error {
	logger = logger.WithField("filePath", filePath)
	handle, err := windows.CreateFile(
		windows.StringToUTF16Ptr(filePath),
		windows.READ_CONTROL|windows.WRITE_DAC,
		windows.FILE_SHARE_READ,
		nil,
		windows.OPEN_EXISTING,
		0,
		0,
	)
	if err != nil {
		logger.WithError(err).Error("open file failed")
		return err
	}
	defer windows.CloseHandle(handle)

	var sd *windows.SECURITY_DESCRIPTOR

	sd, err = windows.GetSecurityInfo(
		windows.Handle(handle),
		windows.SE_FILE_OBJECT,
		windows.DACL_SECURITY_INFORMATION,
	)
	if err != nil {
		logger.WithError(err).Error("get security info failed")
	}

	dacl, _, err := sd.DACL()
	if err != nil {
		logger.WithError(err).Error("get dacl failed")
		return err
	}

	permissionOK := false
	for i := 0; i < int(dacl.AceCount); i++ {
		ace := &windows.ACCESS_ALLOWED_ACE{}
		if err := windows.GetAce(dacl, uint32(i), &ace); err != nil {
			logger.WithError(err).Error("get ace failed")
			return err
		}

		sidptr := uintptr(unsafe.Pointer(&ace.SidStart))
		tsid := (*windows.SID)(unsafe.Pointer(sidptr))

		if tsid.Equals(targetSid) {
			readok := (ace.Mask & FILE_GENERIC_READ) == FILE_GENERIC_READ
			executeok := (ace.Mask & FILE_GENERIC_EXECUTE) == FILE_GENERIC_EXECUTE
			logger.Infof("readok[%v], executeok[%v]", readok, executeok)
			permissionOK = readok && executeok
			break
		}
	}

	if !permissionOK {
		logger.Info("need to grant read and execute right")
		trusteeValue := windows.TrusteeValueFromSID(targetSid)
		addAccess := windows.EXPLICIT_ACCESS{}
		addAccess.AccessPermissions |= windows.GENERIC_READ
		addAccess.AccessPermissions |= windows.GENERIC_EXECUTE
		addAccess.AccessMode = windows.GRANT_ACCESS
		addAccess.Inheritance = windows.NO_INHERITANCE
		addAccess.Trustee = windows.TRUSTEE{
			MultipleTrustee:          nil,
			MultipleTrusteeOperation: windows.NO_MULTIPLE_TRUSTEE,
			TrusteeForm:              windows.TRUSTEE_IS_SID,
			TrusteeType:              windows.TRUSTEE_IS_USER,
			TrusteeValue:             trusteeValue,
		}
		newdacl, err := windows.ACLFromEntries([]windows.EXPLICIT_ACCESS{addAccess}, dacl)
		if err != nil {
			logger.WithError(err).Error("add access to dacl failed")
			return err
		}

		err = windows.SetSecurityInfo(
			windows.Handle(handle),
			windows.SE_FILE_OBJECT,
			windows.DACL_SECURITY_INFORMATION,
			nil,
			nil,
			newdacl,
			nil,
		)
		if err != nil {
			logger.WithError(err).Error("update security info failed")
			return err
		}

	}
	return nil
}

// Environment variable processing occurs in several steps.
//
//	Core system environment variables: ALLUSERSPROFILE, ProgramData, PUBLIC, SystemDrive, SystemRoot.
//	System environment variables of type REG_SZ.
//	System environment variables of type REG_EXPAND_SZ.
//	Core user environment variables: APPDATA, COMPUTERNAME, LOCALAPPDATA, ProgramFiles, USERPROFILE.
//	User environment variables of type REG_SZ.
//	User environment variables of type REG_EXPAND_SZ.
//	Account environment variables: USERDNSDOMAIN, USERDOMAIN, USERNAME.
//
// https://devblogs.microsoft.com/oldnewthing/20231212-00/?p=109137
func setupSessionEnvs(logger logrus.FieldLogger, username string) ([]string, string) {
	var pwd string
	mp := make(map[string]string)
	mpl := make(map[string]string)

	// Disables handle caching of the predefined registry handle for HKEY_CURRENT_USER for the current process.
	// https://stackoverflow.com/questions/1429837/impersonation-to-get-user-hkey-current-user-does-not-work/14947578#14947578
	if err := winapi.RegDisablePredefinedCache(); err != nil {
		logger.WithError(err).Error("regDisablePredefinedCache failed")
	}

	// Inherited core system environment variables and other default environment variables from the parent process
	getDefaultEnvs(logger, &mpl, &mp)
	// Get system environment variables from the registry
	if err := getSysEnvs(logger, &mpl, &mp); err != nil {
		logger.WithError(err).Error("get system envs failed")
	}

	// Set core user environment variables and account environment variables based on user information
	// https://github.com/PowerShell/openssh-portable/blob/v9.8.3.0/contrib/win32/win32compat/w32-doexec.c#L106
	setEnv("USERNAME", username, &mpl, &mp)
	if userInfo, err := user.Lookup(username); err == nil {
		profilePath := userInfo.HomeDir
		setEnv("USERPROFILE", profilePath, &mpl, &mp)
		setEnv("HOME", userInfo.HomeDir, &mpl, &mp)

		if idx := strings.IndexByte(userInfo.HomeDir, ':'); idx > 0 {
			setEnv("HOMEDRIVE", userInfo.HomeDir[:idx+1], &mpl, &mp)
			setEnv("HOMEPATH", userInfo.HomeDir[idx+1:], &mpl, &mp)
		} else {
			setEnv("HOMEPATH", userInfo.HomeDir, &mpl, &mp)
		}

		setEnv("LOCALAPPDATA", fmt.Sprintf("%s\\AppData\\Local", profilePath), &mpl, &mp)
		setEnv("APPDATA", fmt.Sprintf("%s\\AppData\\Roaming", profilePath), &mpl, &mp)
		if fileutil.CheckDirectoryIsExist(userInfo.HomeDir) {
			pwd = userInfo.HomeDir
		} else {
			logger.WithField("homeDir", userInfo.HomeDir).Info("user home directory not exist, do not set pwd")
		}
	}

	// Get User environment variables from the registry
	if err := getUserEnvs(logger, &mpl, &mp); err != nil {
		logger.WithError(err).Error("get user envs failed")
	}

	res := []string{}
	for k, v := range mp {
		res = append(res, fmt.Sprintf("%s=%s", k, v))
	}
	return res, pwd
}

func setEnv(k string, v string, mpl, mp *map[string]string) {
	(*mp)[k] = v
	(*mpl)[strings.ToLower(k)] = v
}

func getDefaultEnvs(logger logrus.FieldLogger, mpl, mp *map[string]string) {
	for _, e := range os.Environ() {
		if idx := strings.IndexByte(e, '='); idx > 0 {
			k := e[:idx]
			lk := strings.ToLower(k)
			if lk == "path" {
				continue
			}
			v := e[idx+1:]
			(*mp)[k] = v
			(*mpl)[lk] = v
		}
	}
}

func getSysEnvs(logger logrus.FieldLogger, mpl, mp *map[string]string) error {
	return getEnvsFromRegistry(logger, registry.LOCAL_MACHINE, `SYSTEM\CurrentControlSet\Control\Session Manager\Environment`, mpl, mp)
}

func getUserEnvs(logger logrus.FieldLogger, mpl, mp *map[string]string) error {
	return getEnvsFromRegistry(logger, registry.CURRENT_USER, `Environment`, mpl, mp)
}

var (
	// Matches other variable names referenced in environment variables
	envReg = regexp.MustCompile(`%[a-zA-Z0-9_]+%`)
)

func getEnvsFromRegistry(logger logrus.FieldLogger, key registry.Key, path string, mpl, mp *map[string]string) error {
	logger = logger.WithFields(logrus.Fields{
		"registryKey":  key,
		"registryPath": path,
	})

	key, err := registry.OpenKey(key, path, registry.QUERY_VALUE|registry.WOW64_64KEY)
	if err != nil {
		logger.WithError(err).Error("Failed to open registry key")
		return err
	}
	defer key.Close()

	names, err := key.ReadValueNames(0)
	if err != nil {
		logger.WithError(err).Error("Failed to read subkey names")
		return err
	}
	for _, k := range names {
		v, valtype, err := key.GetStringValue(k)
		if err != nil {
			logger.WithError(err).Error("Failed to read value of subkey")
			return err
		}
		lk := strings.ToLower(k)
		if valtype == registry.SZ {
			// do nothing
		} else if valtype == registry.EXPAND_SZ {
			// Expand variables of type REG_EXPAND_SZ depend on variables set by a previous step
			if envReg.MatchString(v) {
				v = envReg.ReplaceAllStringFunc(v, func(matched string) string {
					name := strings.ToLower(strings.Trim(matched, "%"))
					if val, ok := (*mpl)[name]; ok {
						return val
					} else {
						return matched
					}
				})
			}
		}
		// The User definition of the PATH environment variable is appended to the System definition, rather than replacing it
		if lk == "path" {
			k = "Path"
			if envPath, ok := (*mp)[k]; ok {
				v = envPath + ";" + v
			}
		}
		(*mp)[k] = v
		(*mpl)[lk] = v
	}
	return nil
}
