package sessionresult

import (
	"errors"
	"fmt"
	"testing"
)

func TestNewOk(t *testing.T) {
	res := NewOk("normal end")
	if res.Code != "Ok" || !res.OK {
		t.Errorf("Expected Code 'Ok' and OK true, got Code='%s', OK=%v", res.Code, res.OK)
	}
	if res.Info != "The session ended normally because normal end" {
		t.Errorf("Expected Info '%s', got '%s'", "The session ended normally because normal end", res.Info)
	}
}

func TestNewNotified(t *testing.T) {
	res := NewNotified()
	if res.Code != "Notified" || !res.OK {
		t.Errorf("Expected Code 'Notified' and OK true, got Code='%s', OK=%v", res.Code, res.OK)
	}
}

func TestNewIdleTimeoutError(t *testing.T) {
	idleTimeoutSecond := 30
	res := NewIdleTimeoutError(idleTimeoutSecond)
	expectedInfo := fmt.Sprintf("The connection is automatically disconnected because the idle time is too long (exceeds %d seconds).", idleTimeoutSecond)
	if res.Code != "IdleTimeout" || res.Info != expectedInfo || !res.OK {
		t.Errorf("Expected Code 'IdleTimeout', Info '%s', and OK true, got Code='%s', Info='%s', OK=%v", expectedInfo, res.Code, res.Info, res.OK)
	}
}

func TestNewSessionTimeoutError(t *testing.T) {
	sessionTimeoutSecond := 60
	res := NewSessionTimeoutError(sessionTimeoutSecond)
	expectedInfo := fmt.Sprintf("The maximum allowed time for a session is %d seconds.", sessionTimeoutSecond)
	if res.Code != "SessionTimeout" || res.Info != expectedInfo || !res.OK {
		t.Errorf("Expected Code 'SessionTimeout', Info '%s', and OK true, got Code='%s', Info='%s', OK=%v", expectedInfo, res.Code, res.Info, res.OK)
	}
}

func TestNewSessionIdDuplicateError(t *testing.T) {
	sessionId := "abc123"
	res := NewSessionIdDuplicateError(sessionId)
	expectedInfo := fmt.Sprintf("Session ID %s is duplicated.", sessionId)
	if res.Code != "SessionIdDuplicate" || res.Info != expectedInfo || res.OK {
		t.Errorf("Expected Code 'SessionIdDuplicate', Info '%s', and OK false, got Code='%s', Info='%s', OK=%v", expectedInfo, res.Code, res.Info, res.OK)
	}
}

func TestNewServerDomainUnavailableError(t *testing.T) {
	res := NewServerDomainUnavailableError()
	expectedInfo := "Unable to obtain the server domain name."
	if res.Code != "ServerDomainUnavailable" || res.Info != expectedInfo || res.OK {
		t.Errorf("Expected Code 'ServerDomainUnavailable', Info '%s', and OK false, got Code='%s', Info='%s', OK=%v", expectedInfo, res.Code, res.Info, res.OK)
	}
}

func TestNewOpenChannelFailedError(t *testing.T) {
	err := errors.New("open failed")
	res := NewOpenChannelFailedError(err)
	expectedInfo := fmt.Sprintf("Failed to open the data channel. %v.", err)
	if res.Code != "OpenChannelFailed" || res.Info != expectedInfo || res.OK {
		t.Errorf("Expected Code 'OpenChannelFailed', Info '%s', and OK false, got Code='%s', Info='%s', OK=%v", expectedInfo, res.Code, res.Info, res.OK)
	}
}

func TestNewMalformedCommandLineError(t *testing.T) {
	cmdline := "invalid command"
	err := errors.New("parse error")
	res := NewMalformedCommandLineError(cmdline, err)
	expectedInfo := fmt.Sprintf("Malformed commandLine `%s` can not be segmented correctly. %v.", cmdline, err)
	if res.Code != "MalformedCommandLine" || res.Info != expectedInfo || res.OK {
		t.Errorf("Expected Code 'MalformedCommandLine', Info '%s', and OK false, got Code='%s', Info='%s', OK=%v", expectedInfo, res.Code, res.Info, res.OK)
	}
}

func TestNewUserNotExistsError(t *testing.T) {
	userName := "testuser"
	res := NewUserNotExistsError(userName)
	expectedInfo := fmt.Sprintf("User %s does not exist.", userName)
	if res.Code != "UserNotExists" || res.Info != expectedInfo || res.OK {
		t.Errorf("Expected Code 'UserNotExists', Info '%s', and OK false, got Code='%s', Info='%s', OK=%v", expectedInfo, res.Code, res.Info, res.OK)
	}
}

func TestNewObtainUserIdentityFailedError(t *testing.T) {
	userName := "testuser"
	err := errors.New("identity fetch failed")
	res := NewObtainUserIdentityFailedError(userName, err)
	expectedInfo := fmt.Sprintf("Failed to obtain identity of user %s: %v.", userName, err)
	if res.Code != "ObtainUserIdentityFailed" || res.Info != expectedInfo || res.OK {
		t.Errorf("Expected Code 'ObtainUserIdentityFailed', Info '%s', and OK false, got Code='%s', Info='%s', OK=%v", expectedInfo, res.Code, res.Info, res.OK)
	}
}

func TestNewObtainUserInfoFailedError(t *testing.T) {
	userName := "testuser"
	err := errors.New("user info fetch failed")
	res := NewObtainUserInfoFailedError(userName, err)
	expectedInfo := fmt.Sprintf("Failed to obtain user information of %s: %v.", userName, err)
	if res.Code != "ObtainUserInfoFailed" || res.Info != expectedInfo || res.OK {
		t.Errorf("Expected Code 'ObtainUserInfoFailed', Info '%s', and OK false, got Code='%s', Info='%s', OK=%v", expectedInfo, res.Code, res.Info, res.OK)
	}
}

func TestNewShellCommandNotFoundError(t *testing.T) {
	shellCmd := "ls"
	err := errors.New("command not found")
	res := NewShellCommandNotFoundError(shellCmd, err)
	expectedInfo := fmt.Sprintf("Look up shell command %s failed: %v.", shellCmd, err)
	if res.Code != "ShellCommandNotFound" || res.Info != expectedInfo || res.OK {
		t.Errorf("Expected Code 'ShellCommandNotFound', Info '%s', and OK false, got Code='%s', Info='%s', OK=%v", expectedInfo, res.Code, res.Info, res.OK)
	}
}

func TestNewShellCommandPermissionDeniedError(t *testing.T) {
	shellCmdPath := "/bin/ls"
	permissions := "0400"
	res := NewShellCommandPermissionDeniedError(shellCmdPath, permissions)
	expectedInfo := fmt.Sprintf("Shell command %s lacks executable permissions: %s.", shellCmdPath, permissions)
	if res.Code != "ShellCommandPermissionDenied" || res.Info != expectedInfo || res.OK {
		t.Errorf("Expected Code 'ShellCommandPermissionDenied', Info '%s', and OK false, got Code='%s', Info='%s', OK=%v", expectedInfo, res.Code, res.Info, res.OK)
	}
}

func TestNewObtainShellCommandPermissionError(t *testing.T) {
	shellCmdPath := "/bin/ls"
	err := errors.New("permission check failed")
	res := NewObtainShellCommandPermissionError(shellCmdPath, err)
	expectedInfo := fmt.Sprintf("An error occurred while obtaining the shell command %s permission. %v.", shellCmdPath, err)
	if res.Code != "ShellCommandPermissionDenied" || res.Info != expectedInfo || res.OK {
		t.Errorf("Expected Code 'ShellCommandPermissionDenied', Info '%s', and OK false, got Code='%s', Info='%s', OK=%v", expectedInfo, res.Code, res.Info, res.OK)
	}
}

func TestNewHomeDirNotFoundError(t *testing.T) {
	homeDir := "/home/testuser"
	res := NewHomeDirNotFoundError(homeDir)
	expectedInfo := fmt.Sprintf("Home directory %s is not found.", homeDir)
	if res.Code != "HomeDirNotFound" || res.Info != expectedInfo || res.OK {
		t.Errorf("Expected Code 'HomeDirNotFound', Info '%s', and OK false, got Code='%s', Info='%s', OK=%v", expectedInfo, res.Code, res.Info, res.OK)
	}
}

func TestNewHomeDirPermissionUnReadableError(t *testing.T) {
	homeDir := "/home/testuser"
	permissions := "0700"
	res := NewHomeDirPermissionUnReadableError(homeDir, permissions)
	expectedInfo := fmt.Sprintf("Home directory %s lacks %s permissions: %s.", homeDir, "readable", permissions)
	if res.Code != "HomeDirPermissionDenied" || res.Info != expectedInfo || res.OK {
		t.Errorf("Expected Code 'HomeDirPermissionDenied', Info '%s', and OK false, got Code='%s', Info='%s', OK=%v", expectedInfo, res.Code, res.Info, res.OK)
	}
}

func TestNewHomeDirPermissionUnExecutableError(t *testing.T) {
	homeDir := "/home/testuser"
	permissions := "0600"
	res := NewHomeDirPermissionUnExecutableError(homeDir, permissions)
	expectedInfo := fmt.Sprintf("Home directory %s lacks %s permissions: %s.", homeDir, "executable", permissions)
	if res.Code != "HomeDirPermissionDenied" || res.Info != expectedInfo || res.OK {
		t.Errorf("Expected Code 'HomeDirPermissionDenied', Info '%s', and OK false, got Code='%s', Info='%s', OK=%v", expectedInfo, res.Code, res.Info, res.OK)
	}
}

func TestNewHomeDirBelongIncorrectUserError(t *testing.T) {
	homeDir := "/home/testuser"
	uid := uint32(1001)
	res := NewHomeDirBelongIncorrectUserError(homeDir, uid)
	expectedInfo := fmt.Sprintf("The home directory %s belongs to an incorrect user %d.", homeDir, uid)
	if res.Code != "HomeDirPermissionDenied" || res.Info != expectedInfo || res.OK {
		t.Errorf("Expected Code 'HomeDirPermissionDenied', Info '%s', and OK false, got Code='%s', Info='%s', OK=%v", expectedInfo, res.Code, res.Info, res.OK)
	}
}

func TestNewObtainHomeDirPermissionError(t *testing.T) {
	homeDir := "/home/testuser"
	err := errors.New("failed to get permissions")
	res := NewObtainHomeDirPermissionError(homeDir, err)
	expectedInfo := fmt.Sprintf("An error occurred while obtaining the home directory %s permission. %v.", homeDir, err)
	if res.Code != "HomeDirPermissionDenied" || res.Info != expectedInfo || res.OK {
		t.Errorf("Expected Code 'HomeDirPermissionDenied', Info '%s', and OK false, got Code='%s', Info='%s', OK=%v", expectedInfo, res.Code, res.Info, res.OK)
	}
}

func TestNewOpenPtyFailedError(t *testing.T) {
	err := errors.New("pty failed")
	res := NewOpenPtyFailedError(err)
	expectedInfo := fmt.Sprintf("An error occurred while starting the PTY. %v.", err)
	if res.Code != "OpenPtyFailed" || res.Info != expectedInfo || res.OK {
		t.Errorf("Expected Code 'OpenPtyFailed', Info '%s', and OK false, got Code='%s', Info='%s', OK=%v", expectedInfo, res.Code, res.Info, res.OK)
	}
}

func TestNewProcessStdoutDataErrorError(t *testing.T) {
	err := errors.New("stdout parse failed")
	res := NewProcessStdoutDataErrorError(err)
	expectedInfo := fmt.Sprintf("An error occurred while parsing shell's stdout. %v.", err)
	if res.Code != "ProcessStdoutDataError" || res.Info != expectedInfo || res.OK {
		t.Errorf("Expected Code 'ProcessStdoutDataError', Info '%s', and OK false, got Code='%s', Info='%s', OK=%v", expectedInfo, res.Code, res.Info, res.OK)
	}
}

func TestNewSendingDataFailedError(t *testing.T) {
	err := errors.New("send failed")
	res := NewSendingDataFailedError(err)
	expectedInfo := fmt.Sprintf("An error occurred while sending data. %v.", err)
	if res.Code != "SendingDataFailed" || res.Info != expectedInfo || res.OK {
		t.Errorf("Expected Code 'SendingDataFailed', Info '%s', and OK false, got Code='%s', Info='%s', OK=%v", expectedInfo, res.Code, res.Info, res.OK)
	}
}

func TestNewKeyExchangeFailedError(t *testing.T) {
	err := errors.New("key exchange failed")
	res := NewKeyExchangeFailedError(err)
	expectedInfo := fmt.Sprintf("An error occurred while exchanging key. %v.", err)
	if res.Code != "KeyExchangeFailed" || res.Info != expectedInfo || res.OK {
		t.Errorf("Expected Code 'SendingDataFailed', Info '%s', and OK false, got Code='%s', Info='%s', OK=%v", expectedInfo, res.Code, res.Info, res.OK)
	}
}

func TestNewOpenTargetPortFailedError(t *testing.T) {
	port := "8080"
	err := errors.New("open failed")
	res := NewOpenTargetPortFailedError(port, err)
	expectedInfo := fmt.Sprintf("An error occurred while opening the target port %s. %v.", port, err)
	if res.Code != "OpenTargetPortFailed" || res.Info != expectedInfo || res.OK {
		fmt.Printf("Expected Code 'OpenTargetPortFailed', Info '%s', and OK false, got Code='%s', Info='%s', OK=%v", expectedInfo, res.Code, res.Info, res.OK)
	}
}

func TestNewReadFromTargetPortFailedError(t *testing.T) {
	port := "8080"
	err := errors.New("read failed")
	res := NewReadFromTargetPortFailedError(port, err)
	expectedInfo := fmt.Sprintf("An error occurred while reading from the target port %s. %v.", port, err)
	if res.Code != "ReadFromTargetPortFailed" || res.Info != expectedInfo || res.OK {
		t.Errorf("Expected Code 'ReadFromTargetPortFailed', Info '%s', and OK false, got Code='%s', Info='%s', OK=%v", expectedInfo, res.Code, res.Info, res.OK)
	}
}

func TestNewReopenTargetPortFailedError(t *testing.T) {
	port := "8080"
	err := errors.New("reopen failed")
	res := NewReopenTargetPortFailedError(port, err)
	expectedInfo := fmt.Sprintf("An error occurred while reopening the target port %s. %v.", port, err)
	if res.Code != "ReopenTargetPortFailed" || res.Info != expectedInfo || res.OK {
		t.Errorf("Expected Code 'ReopenTargetPortFailed', Info '%s', and OK false, got Code='%s', Info='%s', OK=%v", expectedInfo, res.Code, res.Info, res.OK)
	}
}

func TestNewReadFromWebsocketFailedError(t *testing.T) {
	err := errors.New("websocket read failed")
	res := NewReadFromWebsocketFailedError(err)
	expectedInfo := fmt.Sprintf("An error occurred while reading from websocket connection. %v.", err)
	if res.Code != "ReadFromWebsocketFailed" || res.Info != expectedInfo || res.OK {
		t.Errorf("Expected Code 'ReadFromWebsocketFailed', Info '%s', and OK false, got Code='%s', Info='%s', OK=%v", expectedInfo, res.Code, res.Info, res.OK)
	}
}

func TestNewWinptyLoadFailedError(t *testing.T) {
	filename := "winpty.dll"
	err := errors.New("file load failed")
	res := NewWinptyLoadFailedError(filename, err)
	expectedInfo := fmt.Sprintf("%s failed to load. %v.", filename, err)
	if res.Code != "WinptyLoadFailed" || res.Info != expectedInfo || res.OK {
		t.Errorf("Expected Code 'WinptyLoadFailed', Info '%s', and OK false, got Code='%s', Info='%s', OK=%v", expectedInfo, res.Code, res.Info, res.OK)
	}
}

func TestNewUnknownError(t *testing.T) {
	recoverErr := "panic occurred"
	res := NewUnknownError(recoverErr)
	expectedInfo := fmt.Sprintf("Unknown error. %v.", recoverErr)
	if res.Code != "UnknownError" || res.Info != expectedInfo || res.OK {
		t.Errorf("Expected Code 'UnknownError', Info '%s', and OK false, got Code='%s', Info='%s', OK=%v", expectedInfo, res.Code, res.Info, res.OK)
	}
}
