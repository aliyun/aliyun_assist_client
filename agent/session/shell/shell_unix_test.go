package shell

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"reflect"
	"testing"

	// "fmt"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/aliyun/aliyun_assist_client/agent/log"
	"github.com/aliyun/aliyun_assist_client/agent/util/process"
	"github.com/creack/pty"
	"github.com/google/shlex"
	"github.com/stretchr/testify/assert"
)

func TestStartPty(t *testing.T) {

}

func TestStartPty_EmptyCmdContent_DefaultCommandUsed(t *testing.T) {
	mockPlugin := &ShellPlugin{}
	mockPlugin.cmdContent = ""
	mockPlugin.logger = log.GetLogger()

	defer gomonkey.ApplyFunc(process.CreateLocalAdminUser, func(string) error { return nil }).Reset()
	defer gomonkey.ApplyFunc(process.GetUserCredentials, func(string) (uint32, uint32, []uint32, error) {
		return 0, 0, []uint32{0}, nil
	}).Reset()
	defer gomonkey.ApplyFunc(user.Lookup, func(username string) (*user.User, error) {
		return &user.User{
			Username: username,
			Uid:      "0",
			Gid:      "0",
			HomeDir:  "/home/test",
		}, nil
	}).Reset()
	defer gomonkey.ApplyFunc(pty.Start, func(cmd *exec.Cmd) (*os.File, error) {
		assert.Equal(t, []string{shellCommand}, cmd.Args)
		return nil, nil
	}).Reset()
	defer gomonkey.ApplyMethod(reflect.TypeOf(mockPlugin), "SetSize", func(*ShellPlugin, uint32, uint32) error {
		return nil
	}).Reset()
	os.MkdirAll("/home/test", 0755)
	defer os.RemoveAll("/home/test")

	err := StartPty(mockPlugin)
	assert.Nil(t, err)

}

func TestStartPty_NonEmptyCmdContent_DefaultCommandUsed(t *testing.T) {
	mockPlugin := &ShellPlugin{}
	mockPlugin.cmdContent = "ls -lh /root"
	mockPlugin.logger = log.GetLogger()

	defer gomonkey.ApplyFunc(process.CreateLocalAdminUser, func(string) error { return nil }).Reset()
	defer gomonkey.ApplyFunc(process.GetUserCredentials, func(string) (uint32, uint32, []uint32, error) {
		return 0, 0, []uint32{0}, nil
	}).Reset()
	defer gomonkey.ApplyFunc(user.Lookup, func(username string) (*user.User, error) {
		return &user.User{
			Username: username,
			Uid:      "0",
			Gid:      "0",
			HomeDir:  "/home/test",
		}, nil
	}).Reset()
	defer gomonkey.ApplyFunc(pty.Start, func(cmd *exec.Cmd) (*os.File, error) {
		cmdArgs, err := shlex.Split(mockPlugin.cmdContent)
		assert.Nil(t, err)
		assert.Equal(t, cmdArgs, cmd.Args)
		return nil, nil
	}).Reset()
	defer gomonkey.ApplyMethod(reflect.TypeOf(mockPlugin), "SetSize", func(*ShellPlugin, uint32, uint32) error {
		return nil
	}).Reset()
	os.MkdirAll("/home/test", 0755)
	defer os.RemoveAll("/home/test")

	sessionRes := StartPty(mockPlugin)
	assert.Nil(t, sessionRes)

	cmdArgs, err := shlex.Split(mockPlugin.cmdContent)
	assert.Nil(t, err)
	assert.Equal(t, "/usr/bin/ls", mockPlugin.cmd.Path)
	assert.Equal(t, cmdArgs, mockPlugin.cmd.Args)
}

func TestStartPty_EmptyUsername_LocalAdminCreated(t *testing.T) {
	mockPlugin := &ShellPlugin{}
	mockPlugin.cmdContent = ""
	mockPlugin.logger = log.GetLogger()

	var createLocalAdminUserName string
	var getUserCredentialsName string

	defer gomonkey.ApplyFunc(process.CreateLocalAdminUser, func(name string) error {
		createLocalAdminUserName = name
		return nil
	}).Reset()
	defer gomonkey.ApplyFunc(process.GetUserCredentials, func(name string) (uint32, uint32, []uint32, error) {
		getUserCredentialsName = name
		return 0, 0, []uint32{0}, nil
	}).Reset()
	defer gomonkey.ApplyFunc(user.Lookup, func(username string) (*user.User, error) {
		return &user.User{
			Username: username,
			Uid:      "0",
			Gid:      "0",
			HomeDir:  "/home/test",
		}, nil
	}).Reset()
	defer gomonkey.ApplyFunc(pty.Start, func(*exec.Cmd) (*os.File, error) {
		return nil, nil
	}).Reset()
	defer gomonkey.ApplyMethod(reflect.TypeOf(mockPlugin), "SetSize", func(*ShellPlugin, uint32, uint32) error {
		return nil
	}).Reset()
	os.MkdirAll("/home/test", 0755)
	defer os.RemoveAll("/home/test")

	err := StartPty(mockPlugin)
	assert.Nil(t, err)

	assert.Equal(t, default_runas_user, createLocalAdminUserName)
	assert.Equal(t, default_runas_user, getUserCredentialsName)
}

func TestStartPty_NonExistentUser_ErrorReturned(t *testing.T) {
	mockPlugin := &ShellPlugin{}
	mockPlugin.cmdContent = ""
	mockPlugin.username = "non-existent-user"
	mockPlugin.logger = log.GetLogger()

	defer gomonkey.ApplyFunc(process.CreateLocalAdminUser, func(name string) error {
		return nil
	}).Reset()
	defer gomonkey.ApplyFunc(process.DoesUserExist, func(name string) (bool, error) {
		return false, nil
	}).Reset()
	defer gomonkey.ApplyFunc(process.GetUserCredentials, func(name string) (uint32, uint32, []uint32, error) {
		return 0, 0, []uint32{0}, nil
	}).Reset()
	defer gomonkey.ApplyFunc(user.Lookup, func(username string) (*user.User, error) {
		return &user.User{
			Username: username,
			Uid:      "0",
			Gid:      "0",
			HomeDir:  "/home/test",
		}, nil
	}).Reset()
	defer gomonkey.ApplyFunc(pty.Start, func(*exec.Cmd) (*os.File, error) {
		return nil, nil
	}).Reset()
	defer gomonkey.ApplyMethod(reflect.TypeOf(mockPlugin), "SetSize", func(*ShellPlugin, uint32, uint32) error {
		return nil
	}).Reset()
	os.MkdirAll("/home/test", 0755)
	defer os.RemoveAll("/home/test")

	err := StartPty(mockPlugin)
	assert.NotNil(t, err)
	assert.Equal(t, "UserNotExists", err.Code)
}

func TestStartPty_UserExists_CredentialsRetrieved(t *testing.T) {
	mockPlugin := &ShellPlugin{}
	mockPlugin.cmdContent = ""
	mockPlugin.username = "existent-user"
	mockPlugin.logger = log.GetLogger()

	defer gomonkey.ApplyFunc(process.CreateLocalAdminUser, func(name string) error {
		return nil
	}).Reset()
	defer gomonkey.ApplyFunc(process.DoesUserExist, func(name string) (bool, error) {
		return true, nil
	}).Reset()
	defer gomonkey.ApplyFunc(process.GetUserCredentials, func(name string) (uint32, uint32, []uint32, error) {
		return 0, 0, []uint32{0}, errors.New("GetUserCredentials error")
	}).Reset()
	defer gomonkey.ApplyFunc(user.Lookup, func(username string) (*user.User, error) {
		return &user.User{
			Username: username,
			Uid:      "0",
			Gid:      "0",
			HomeDir:  "/home/test",
		}, nil
	}).Reset()
	defer gomonkey.ApplyFunc(pty.Start, func(*exec.Cmd) (*os.File, error) {
		return nil, nil
	}).Reset()
	defer gomonkey.ApplyMethod(reflect.TypeOf(mockPlugin), "SetSize", func(*ShellPlugin, uint32, uint32) error {
		return nil
	}).Reset()

	err := StartPty(mockPlugin)
	assert.NotNil(t, err)
	assert.Equal(t, "ObtainUserIdentityFailed", err.Code)
}

func TestStartPty_NonZeroWindowSize_TerminalSizeSet(t *testing.T) {
	mockPlugin := &ShellPlugin{}
	mockPlugin.cmdContent = ""
	mockPlugin.logger = log.GetLogger()

	var setSizeCalled bool
	defer gomonkey.ApplyFunc(process.CreateLocalAdminUser, func(name string) error {
		return nil
	}).Reset()
	defer gomonkey.ApplyFunc(process.GetUserCredentials, func(name string) (uint32, uint32, []uint32, error) {
		return 0, 0, []uint32{0}, nil
	}).Reset()
	defer gomonkey.ApplyFunc(user.Lookup, func(username string) (*user.User, error) {
		return &user.User{
			Username: username,
			Uid:      "0",
			Gid:      "0",
			HomeDir:  "/home/test",
		}, nil
	}).Reset()
	defer gomonkey.ApplyFunc(pty.Start, func(*exec.Cmd) (*os.File, error) {
		return nil, nil
	}).Reset()
	defer gomonkey.ApplyMethod(reflect.TypeOf(mockPlugin), "SetSize", func(*ShellPlugin, uint32, uint32) error {
		setSizeCalled = true
		return nil
	}).Reset()
	os.MkdirAll("/home/test", 0755)
	defer os.RemoveAll("/home/test")

	err := StartPty(mockPlugin)
	assert.Nil(t, err)
	assert.False(t, setSizeCalled)

	mockPlugin.first_ws_col = 100
	err = StartPty(mockPlugin)
	assert.Nil(t, err)
	assert.True(t, setSizeCalled)
}

func TestStartPty_StartFailed(t *testing.T) {
	mockPlugin := &ShellPlugin{}
	mockPlugin.cmdContent = ""
	mockPlugin.logger = log.GetLogger()

	defer gomonkey.ApplyFunc(process.CreateLocalAdminUser, func(name string) error {
		return nil
	}).Reset()
	defer gomonkey.ApplyFunc(process.GetUserCredentials, func(name string) (uint32, uint32, []uint32, error) {
		return 0, 0, []uint32{0}, nil
	}).Reset()
	defer gomonkey.ApplyFunc(user.Lookup, func(username string) (*user.User, error) {
		return &user.User{
			Username: username,
			Uid:      "0",
			Gid:      "0",
			HomeDir:  "/home/test",
		}, nil
	}).Reset()
	defer gomonkey.ApplyFunc(pty.Start, func(*exec.Cmd) (*os.File, error) {
		return nil, errors.New("startpty failed")
	}).Reset()
	defer gomonkey.ApplyMethod(reflect.TypeOf(mockPlugin), "SetSize", func(*ShellPlugin, uint32, uint32) error {
		return nil
	}).Reset()
	os.MkdirAll("/home/test", 0755)
	defer os.RemoveAll("/home/test")

	err := StartPty(mockPlugin)
	assert.NotNil(t, err)
	fmt.Println("StartPty", err)
	assert.Equal(t, "OpenPtyFailed", err.Code)
	assert.Contains(t, err.Error(), "startpty failed")
}
