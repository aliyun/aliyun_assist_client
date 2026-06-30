//go:build linux || freebsd
// +build linux freebsd

package shell

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"syscall"

	"github.com/creack/pty"
	"github.com/google/shlex"

	sessionresult "github.com/aliyun/aliyun_assist_client/agent/session/sessionresult"
	"github.com/aliyun/aliyun_assist_client/agent/util/process"
	"github.com/aliyun/aliyun_assist_client/common/executil"
	"github.com/aliyun/aliyun_assist_client/common/fileutil"
)

type ShellPlugin struct {
	ShellPluginBase
	cmd *exec.Cmd
}

const (
	termEnvVariable    = "TERM=xterm-256color"
	langEnvVariable    = "LANG=C.UTF-8"
	langEnvVariableKey = "LANG"
	homeEnvVariable    = "HOME=/home/"
	default_runas_user = "ecs-assist-user"
)

func StartPty(plugin *ShellPlugin) *sessionresult.SessionResult {
	if plugin.cmdContent == "" {
		plugin.cmd = executil.Command(shellCommand)
	} else {
		cmdArgs, err := shlex.Split(plugin.cmdContent)
		if err != nil {
			return sessionresult.NewMalformedCommandLineError(plugin.cmdContent, err)
		}
		plugin.cmd = executil.Command(cmdArgs[0], cmdArgs[1:]...)
	}

	plugin.cmd.Env = append(os.Environ(), termEnvVariable)

	langEnvVariableValue := os.Getenv(langEnvVariableKey)
	if langEnvVariableValue == "" {
		plugin.cmd.Env = append(plugin.cmd.Env, langEnvVariable)
	}

	default_user := default_runas_user

	if plugin.username == "" {
		process.CreateLocalAdminUser(default_runas_user)
	} else {
		default_user = plugin.username
		if userExists, _ := process.DoesUserExist(plugin.username); !userExists {
			// if user does not exist, fail the session
			return sessionresult.NewUserNotExistsError(plugin.username)
		}
	}

	uid, gid, groups, err := process.GetUserCredentials(default_user)
	if err != nil {
		return sessionresult.NewObtainUserIdentityFailedError(default_user, err)
	}
	plugin.cmd.SysProcAttr = &syscall.SysProcAttr{}
	plugin.cmd.SysProcAttr.Credential = &syscall.Credential{Uid: uid, Gid: gid, Groups: groups, NoSetGroups: false}

	// Setting home environment variable for RunAs user
	userInfo, err := user.Lookup(default_user)
	if err != nil {
		return sessionresult.NewObtainUserInfoFailedError(default_user, err)
	}
	plugin.logger.Infof("Home directory of user `%s`: %s", default_user, userInfo.HomeDir)
	runAsUserHomeEnvVariable := fmt.Sprintf("HOME=%s", userInfo.HomeDir)
	plugin.cmd.Env = append(plugin.cmd.Env, runAsUserHomeEnvVariable)
	plugin.cmd.Dir = userInfo.HomeDir

	// 检查homedir
	if !fileutil.CheckFileIsExist(userInfo.HomeDir) {
		return sessionresult.NewHomeDirNotFoundError(userInfo.HomeDir)
	}
	if err := checkHomeDirPerm(userInfo.HomeDir, uid); err != nil {
		return err
	}

	if plugin.cmdContent == "" {
		if err := checkShellCmd(shellCommand); err != nil {
			return err
		}
	}

	ptyFile, err := pty.Start(plugin.cmd)
	if err != nil {
		plugin.logger.Errorf("Failed to start pty: %s\n", err)
		return sessionresult.NewOpenPtyFailedError(err)
	}
	plugin.stdin = ptyFile
	plugin.stdout = ptyFile

	if plugin.first_ws_col != 0 {
		plugin.SetSize(plugin.first_ws_col, plugin.first_ws_row)
	}

	return nil
}

func (p *ShellPlugin) waitPid() {
	go func() {
		defer func() {
			p.logger.Infoln("stop in run waitPid")

			if err := recover(); err != nil {
				p.logger.Errorf("Error occurred while executing plugin %s: \n%v", p.id, err)
			}
		}()

		p.cmd.Process.Kill()
		p.cmd.Wait()
	}()
}

func (p *ShellPlugin) stop() (err error) {
	p.logger.Info("Stopping pty")
	if p.stdin == nil {
		return nil
	}
	if err := p.stdin.Close(); err != nil {
		if err, ok := err.(*os.PathError); ok && err.Err != os.ErrClosed {
			return err
		}
	}
	return nil
}

func (p *ShellPlugin) SetSize(ws_col, ws_row uint32) (err error) {
	// pty未创建时，先缓存窗口大小
	if p.stdin == nil {
		p.first_ws_col = ws_col
		p.first_ws_row = ws_row
		return nil
	}

	winSize := pty.Winsize{
		Cols: uint16(ws_col),
		Rows: uint16(ws_row),
	}

	if err := pty.Setsize(p.stdin, &winSize); err != nil {
		p.logger.Errorf("set pty size failed: %s", err)
		return err
	}
	return nil
}

func (p *ShellPlugin) onInputStreamData(payload []byte) error {
	if _, err := p.stdin.Write(payload); err != nil {
		p.logger.Errorf("Unable to write to stdin, err: %v.", err)
		return err
	}
	return nil
}

func checkHomeDirPerm(path string, expectUid uint32) *sessionresult.SessionResult {
	fileInfo, err := os.Stat(path)
	if err != nil {
		return sessionresult.NewObtainHomeDirPermissionError(path, err)
	}
	if stat, ok := fileInfo.Sys().(*syscall.Stat_t); ok {
		if stat.Uid != expectUid {
			return sessionresult.NewHomeDirBelongIncorrectUserError(path, stat.Uid)
		}
	}
	mode := fileInfo.Mode()
	if mode.Perm()&0400 == 0 {
		return sessionresult.NewHomeDirPermissionUnReadableError(path, fmt.Sprintf("%o", mode.Perm()))
	}
	if mode.Perm()&0100 == 0 {
		return sessionresult.NewHomeDirPermissionUnExecutableError(path, fmt.Sprintf("%o", mode.Perm()))
	}
	return nil
}

func checkShellCmd(path string) *sessionresult.SessionResult {
	absPath, err := exec.LookPath(path)
	if err != nil && !errors.Is(err, exec.ErrDot) {
		return sessionresult.NewShellCommandNotFoundError(path, err)
	}

	fileInfo, err := os.Stat(absPath)
	if err != nil {
		return sessionresult.NewObtainShellCommandPermissionError(absPath, err)
	}

	if fileInfo.Mode().Perm()&0100 == 0 {
		return sessionresult.NewShellCommandPermissionDeniedError(absPath, fmt.Sprintf("%o", fileInfo.Mode().Perm()))
	}
	return nil
}
