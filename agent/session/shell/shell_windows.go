package shell

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/aliyun/aliyun_assist_client/agent/session/channel"
	"github.com/aliyun/aliyun_assist_client/agent/session/winpty"
	"github.com/aliyun/aliyun_assist_client/common/pathutil"
	"github.com/aliyun/aliyun_assist_client/thirdparty/sirupsen/logrus"
)

const (
	defaultConsoleCol = 200
	defaultConsoleRow = 60
)

type ShellPlugin struct {
	id           string
	stdin        *os.File
	stdout       *os.File
	pty          *winpty.WinPTY
	cmdContent   string
	username     string
	passwordName string
	dataChannel  channel.ISessionChannel
	first_ws_col uint32
	first_ws_row uint32
	sendInterval int

	exitCtx  context.Context
	exitFunc context.CancelCauseFunc

	logger logrus.FieldLogger
}

func StartPty(plugin *ShellPlugin) (err error) {
	finalCmd := "powershell.exe"
	if plugin.cmdContent != "" {
		finalCmd = plugin.cmdContent
	}
	plugin.logger.Infoln("finalCmd ", finalCmd)
	exe_path, err := pathutil.GetExecutableDir()
	if err != nil {
		return fmt.Errorf("failed to determine executable directory: %w", err)
	}

	winptyDllFilePath := filepath.Join(exe_path, "plugin", "SessionManager", "winpty.dll")
	var pty *winpty.WinPTY
	if plugin.username == "" {
		pty, err = winpty.Start(winptyDllFilePath, finalCmd, defaultConsoleCol, defaultConsoleRow, winpty.DEFAULT_WINPTY_FLAGS)
		if err != nil {
			plugin.logger.Errorln("error in winpty.Start")
			return err
		}

	} else {
		plugin.logger.Errorln("not support")
		return fmt.Errorf("not support")
	}

	plugin.pty = pty
	plugin.stdin = pty.StdIn
	plugin.stdout = pty.StdOut

	if plugin.first_ws_col != 0 {
		plugin.SetSize(plugin.first_ws_col, plugin.first_ws_row)
	}
	return err
}

func (p *ShellPlugin) startPtyAsUser(user string, pass string, shellCmd string) (err error) {

	return nil
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
