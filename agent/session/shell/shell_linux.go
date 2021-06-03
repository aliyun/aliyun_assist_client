package shell

import (
	"encoding/json"
	"github.com/aliyun/aliyun_assist_client/agent/log"
	"github.com/aliyun/aliyun_assist_client/agent/session/message"
	"github.com/aliyun/aliyun_assist_client/agent/util/process"
	"github.com/aliyun/aliyun_assist_client/agent/session/channel"
	"os/exec"
	"fmt"
	"os"
	"github.com/creack/pty"
	"strings"
	"syscall"
)

type ShellPlugin struct {
	id        string
	stdin       *os.File
	stdout      *os.File
	cmdContent        string
	username   string
	passwordName string
	dataChannel channel.ISessionChannel
	cmd *exec.Cmd
	first_ws_col uint32
	first_ws_row uint32
}

const (
	termEnvVariable       = "TERM=xterm-256color"
	langEnvVariable       = "LANG=C.UTF-8"
	langEnvVariableKey    = "LANG"
	homeEnvVariable       = "HOME=/home/"
	default_runas_user    = "ecs-assist-user"
)

func StartPty(plugin *ShellPlugin)( err error) {
	if plugin.cmdContent == "" {
		plugin.cmd = exec.Command("bash")
	} else {
		cmdArgs := strings.Split(plugin.cmdContent," ")
		plugin.cmd = exec.Command(cmdArgs[0], cmdArgs[1:]...)
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
			return fmt.Errorf("failed to start pty since RunAs user %s does not exist", plugin.username)
		}
	}

	uid, gid, groups, err := process.GetUserCredentials(default_user)
	if err != nil {
		return err
	}
	plugin.cmd.SysProcAttr = &syscall.SysProcAttr{}
	plugin.cmd.SysProcAttr.Credential = &syscall.Credential{Uid: uid, Gid: gid, Groups: groups, NoSetGroups: false}

	// Setting home environment variable for RunAs user
	runAsUserHomeEnvVariable := homeEnvVariable + default_user
	plugin.cmd.Env = append(plugin.cmd.Env, runAsUserHomeEnvVariable)

	ptyFile, err := pty.Start(plugin.cmd)
	if err != nil {
		log.GetLogger().Errorf("Failed to start pty: %s\n", err)
		return fmt.Errorf("Failed to start pty: %s\n", err)
	}
	plugin.stdin = ptyFile
	plugin.stdout = ptyFile

	if plugin.first_ws_col != 0 {
		plugin.SetSize(plugin.first_ws_col, plugin.first_ws_row)
	}

	return nil
}

func (p *ShellPlugin) waitPid() () {
	go func() {
		defer func() {
			log.GetLogger().Infoln("stop in run waitPid")

			if err := recover(); err != nil {
				log.GetLogger().Errorf("Error occurred while executing plugin %s: \n%v", p.id, err)
			}
		}()

		p.cmd.Process.Kill()
		p.cmd.Wait()
	} ()
}

func (p *ShellPlugin) stop() (err error) {
	log.GetLogger().Info("Stopping pty")
	if p.stdin == nil {
		return nil
	}
	if err := p.stdin.Close(); err != nil {
		if err, ok := err.(*os.PathError); ok && err.Err != os.ErrClosed {
			return fmt.Errorf("unable to close ptyFile. %s", err)
		}
	}
	return nil
}



func (p *ShellPlugin) SetSize(ws_col, ws_row uint32) (err error) {
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
		log.GetLogger().Errorf("set pty size failed: %s", err)
		return fmt.Errorf("set pty size failed: %s", err)
	}
	return nil
}

func (p *ShellPlugin) InputStreamMessageHandler(streamDataMessage message.Message) error {
	log.GetLogger().Println("InputStreamMessageHandler")
	if p.stdin == nil || p.stdout == nil {
		// This is to handle scenario when cli/console starts sending size data but pty has not been started yet
		// Since packets are rejected, cli/console will resend these packets until pty starts successfully in separate thread
		log.GetLogger().Tracef("Pty unavailable. Reject incoming message packet")
		return nil
	}

	switch streamDataMessage.MessageType {
	case message.InputStreamDataMessage:
		// log.GetLogger().Traceln("Input message received: ", streamDataMessage.Payload)
		if _, err := p.stdin.Write(streamDataMessage.Payload); err != nil {
			log.GetLogger().Errorf("Unable to write to stdin, err: %v.", err)
			return err
		}
		break
	case message.SetSizeDataMessage:
		var size SizeData
		if err := json.Unmarshal(streamDataMessage.Payload, &size); err != nil {
			log.GetLogger().Errorf("Invalid size message: %s", err)
			return err
		}
		// log.GetLogger().Tracef("Resize data received: cols: %d, rows: %d", size.Cols, size.Rows)
		if err := p.SetSize(size.Cols, size.Rows); err != nil {
			log.GetLogger().Errorf("Unable to set pty size: %s", err)
			return err
		}
		break
	}
	return nil
}