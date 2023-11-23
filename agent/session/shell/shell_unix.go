//go:build linux || freebsd
// +build linux freebsd

package shell

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"syscall"

	"github.com/creack/pty"
	"github.com/google/shlex"

	"github.com/aliyun/aliyun_assist_client/agent/log"
	"github.com/aliyun/aliyun_assist_client/agent/session/channel"
	"github.com/aliyun/aliyun_assist_client/agent/session/message"
	"github.com/aliyun/aliyun_assist_client/agent/util/process"
	"github.com/aliyun/aliyun_assist_client/common/executil"
)

type ShellPlugin struct {
	id           string
	stdin        *os.File
	stdout       *os.File
	cmdContent   string
	username     string
	passwordName string
	dataChannel  channel.ISessionChannel
	cmd          *exec.Cmd
	first_ws_col uint32
	first_ws_row uint32
	flowLimit    int
	sendInterval int
}

const (
	termEnvVariable    = "TERM=xterm-256color"
	langEnvVariable    = "LANG=C.UTF-8"
	langEnvVariableKey = "LANG"
	homeEnvVariable    = "HOME=/home/"
	default_runas_user = "ecs-assist-user"
)

func StartPty(plugin *ShellPlugin) (err error) {
	if plugin.cmdContent == "" {
		plugin.cmd = executil.Command(shellCommand)
	} else {
		cmdArgs, err := shlex.Split(plugin.cmdContent)
		if err != nil {
			return fmt.Errorf("split command content failed: ", err)
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
	userInfo, err := user.Lookup(default_user)
	if err != nil {
		return err
	}
	log.GetLogger().Infof("Home directory of user `%s`: %s", default_user, userInfo.HomeDir)
	runAsUserHomeEnvVariable := fmt.Sprintf("HOME=%s", userInfo.HomeDir)
	plugin.cmd.Env = append(plugin.cmd.Env, runAsUserHomeEnvVariable)
	plugin.cmd.Dir = userInfo.HomeDir

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

func (p *ShellPlugin) waitPid() {
	go func() {
		defer func() {
			log.GetLogger().Infoln("stop in run waitPid")

			if err := recover(); err != nil {
				log.GetLogger().Errorf("Error occurred while executing plugin %s: \n%v", p.id, err)
			}
		}()

		p.cmd.Process.Kill()
		p.cmd.Wait()
	}()
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
	case message.StatusDataMessage:
		if len(streamDataMessage.Payload) > 0 {
			code, err := message.BytesToIntU(streamDataMessage.Payload[0:1])
			if err == nil {
				if code == 7 { // 设置agent的发送速率
					speed, err := message.BytesToIntU(streamDataMessage.Payload[1:]) // speed 单位是 bps
					if speed == 0 {
						break
					}
					if err != nil {
						log.GetLogger().Errorf("Invalid flowLimit: %s", err)
						return err
					}
					p.sendInterval = 1000 / (speed / 8 / sendPackageSize)
					log.GetLogger().Infof("Set send speed, channelId[%s] speed[%d]bps sendInterval[%d]ms\n", p.id, speed, p.sendInterval)
				}
			} else {
				log.GetLogger().Errorf("Parse status code err: %s", err)
			}
		}
		break
	}
	return nil
}
