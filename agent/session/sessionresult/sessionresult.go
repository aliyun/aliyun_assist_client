package sessionresult

import (
	"fmt"
)

const (
	Ok = "Ok"
)

type SessionResult struct {
	Code string
	Info string
	OK   bool // Is the session ending as expected
}

func (se *SessionResult) Error() string {
	if se.Info == "" {
		return se.Code
	}
	return se.Code + ": " + se.Info
}

// It's OK if the session ended because of normal reasons, notification, idle timeout, session timeout, etc.
func NewOk(reason string) (res *SessionResult) {
	res = &SessionResult{
		Code: Ok,
		OK:   true,
	}
	if reason != "" {
		res.Info = "The session ended normally because " + reason
	}
	return
}

func NewNotified() *SessionResult {
	return &SessionResult{
		Code: "Notified",
		OK:   true,
	}
}

func NewIdleTimeoutError(idleTimeoutSecond int) *SessionResult {
	return &SessionResult{
		Code: "IdleTimeout",
		Info: fmt.Sprintf("The connection is automatically disconnected because the idle time is too long (exceeds %d seconds).", idleTimeoutSecond),
		OK:   true,
	}
}

func NewSessionTimeoutError(sessionTimeoutSecond int) *SessionResult {
	return &SessionResult{
		Code: "SessionTimeout",
		Info: fmt.Sprintf("The maximum allowed time for a session is %d seconds.", sessionTimeoutSecond),
		OK:   true,
	}
}

// It's not OK if the session ended because of other errors.
func NewSessionIdDuplicateError(sessionId string) *SessionResult {
	return &SessionResult{
		Code: "SessionIdDuplicate",
		Info: fmt.Sprintf("Session ID %s is duplicated.", sessionId),
	}
}

func NewServerDomainUnavailableError() *SessionResult {
	return &SessionResult{
		Code: "ServerDomainUnavailable",
		Info: "Unable to obtain the server domain name.",
	}
}

func NewOpenChannelFailedError(err error) *SessionResult {
	return &SessionResult{
		Code: "OpenChannelFailed",
		Info: fmt.Sprintf("Failed to open the data channel. %v.", err),
	}
}

func NewMalformedCommandLineError(cmdline string, err error) *SessionResult {
	return &SessionResult{
		Code: "MalformedCommandLine",
		Info: fmt.Sprintf("Malformed commandLine `%s` can not be segmented correctly. %v.", cmdline, err),
	}
}

func NewUserNotExistsError(userName string) *SessionResult {
	return &SessionResult{
		Code: "UserNotExists",
		Info: fmt.Sprintf("User %s does not exist.", userName),
	}
}

func NewObtainUserIdentityFailedError(userName string, err error) *SessionResult {
	return &SessionResult{
		Code: "ObtainUserIdentityFailed",
		Info: fmt.Sprintf("Failed to obtain identity of user %s: %v.", userName, err),
	}
}

func NewObtainUserInfoFailedError(userName string, err error) *SessionResult {
	return &SessionResult{
		Code: "ObtainUserInfoFailed",
		Info: fmt.Sprintf("Failed to obtain user information of %s: %v.", userName, err),
	}
}

func NewShellCommandNotFoundError(shellCmd string, err error) *SessionResult {
	return &SessionResult{
		Code: "ShellCommandNotFound",
		Info: fmt.Sprintf("Look up shell command %s failed: %v.", shellCmd, err),
	}
}

func NewShellCommandPermissionDeniedError(shellCmdPath, actualPermissions string) *SessionResult {
	return &SessionResult{
		Code: "ShellCommandPermissionDenied",
		Info: fmt.Sprintf("Shell command %s lacks executable permissions: %s.", shellCmdPath, actualPermissions),
	}
}

func NewObtainShellCommandPermissionError(shellCmdPath string, err error) *SessionResult {
	return &SessionResult{
		Code: "ShellCommandPermissionDenied",
		Info: fmt.Sprintf("An error occurred while obtaining the shell command %s permission. %v.", shellCmdPath, err),
	}
}

func NewHomeDirNotFoundError(homeDir string) *SessionResult {
	return &SessionResult{
		Code: "HomeDirNotFound",
		Info: fmt.Sprintf("Home directory %s is not found.", homeDir),
	}
}

func NewHomeDirPermissionUnReadableError(homeDir, actualPermissions string) *SessionResult {
	return &SessionResult{
		Code: "HomeDirPermissionDenied",
		Info: fmt.Sprintf("Home directory %s lacks %s permissions: %s.", homeDir, "readable", actualPermissions),
	}
}

func NewHomeDirPermissionUnExecutableError(homeDir, actualPermissions string) *SessionResult {
	return &SessionResult{
		Code: "HomeDirPermissionDenied",
		Info: fmt.Sprintf("Home directory %s lacks %s permissions: %s.", homeDir, "executable", actualPermissions),
	}
}

func NewHomeDirBelongIncorrectUserError(homeDir string, actualUid uint32) *SessionResult {
	return &SessionResult{
		Code: "HomeDirPermissionDenied",
		Info: fmt.Sprintf("The home directory %s belongs to an incorrect user %d.", homeDir, actualUid),
	}
}

func NewObtainHomeDirPermissionError(homeDir string, err error) *SessionResult {
	return &SessionResult{
		Code: "HomeDirPermissionDenied",
		Info: fmt.Sprintf("An error occurred while obtaining the home directory %s permission. %v.", homeDir, err),
	}
}

func NewOpenPtyFailedError(err error) *SessionResult {
	return &SessionResult{
		Code: "OpenPtyFailed",
		Info: fmt.Sprintf("An error occurred while starting the PTY. %v.", err),
	}
}
func NewProcessStdoutDataErrorError(err error) *SessionResult {
	return &SessionResult{
		Code: "ProcessStdoutDataError",
		Info: fmt.Sprintf("An error occurred while parsing shell's stdout. %v.", err),
	}
}
func NewSendingDataFailedError(err error) *SessionResult {
	return &SessionResult{
		Code: "SendingDataFailed",
		Info: fmt.Sprintf("An error occurred while sending data. %v.", err),
	}
}

func NewKeyExchangeFailedError(err error) *SessionResult {
	return &SessionResult{
		Code: "KeyExchangeFailed",
		Info: fmt.Sprintf("An error occurred while exchanging key. %v.", err),
	}
}

func NewOpenTargetPortFailedError(targetPort string, err error) *SessionResult {
	return &SessionResult{
		Code: "OpenTargetPortFailed",
		Info: fmt.Sprintf("An error occurred while opening the target port %s. %v.", targetPort, err),
	}
}

func NewReadFromTargetPortFailedError(targetPort string, err error) *SessionResult {
	return &SessionResult{
		Code: "ReadFromTargetPortFailed",
		Info: fmt.Sprintf("An error occurred while reading from the target port %s. %v.", targetPort, err),
	}
}

func NewReopenTargetPortFailedError(targetPort string, err error) *SessionResult {
	return &SessionResult{
		Code: "ReopenTargetPortFailed",
		Info: fmt.Sprintf("An error occurred while reopening the target port %s. %v.", targetPort, err),
	}
}

func NewReadFromWebsocketFailedError(err error) *SessionResult {
	return &SessionResult{
		Code: "ReadFromWebsocketFailed",
		Info: fmt.Sprintf("An error occurred while reading from websocket connection. %v.", err),
	}
}

func NewUnknownError(recoverErr any) *SessionResult {
	return &SessionResult{
		Code: "UnknownError",
		Info: fmt.Sprintf("Unknown error. %v.", recoverErr),
	}
}

func NewWinptyLoadFailedError(filename string, err error) *SessionResult {
	return &SessionResult{
		Code: "WinptyLoadFailed",
		Info: fmt.Sprintf("%s failed to load. %v.", filename, err),
	}
}

func NewWinptyPermissionDeniedError(user, filePath string, err error) *SessionResult {
	return &SessionResult{
		Code: "WinptyPermissionDenied",
		Info: fmt.Sprintf("Failed to add user %s readable and executable permissions to %s. %v.", user, filePath, err),
	}
}

func NewPasswordNameNotFoundError(passwordName string, err error) *SessionResult {
	return &SessionResult{
		Code: "PasswordNameNotFound",
		Info: fmt.Sprintf("Password name %s not found. %v.", passwordName, err),
	}
}
