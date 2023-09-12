package shell

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/aliyun/aliyun_assist_client/agent/log"
	"github.com/aliyun/aliyun_assist_client/agent/session/channel"
	"github.com/aliyun/aliyun_assist_client/agent/session/message"
	"github.com/aliyun/aliyun_assist_client/agent/session/winpty"
	"github.com/aliyun/aliyun_assist_client/common/pathutil"
)

const (
	defaultConsoleCol                                = 200
	defaultConsoleRow                                = 60
)

type ShellPlugin struct {
	id        string
	stdin       *os.File
	stdout      *os.File
	pty *winpty.WinPTY
	cmdContent        string
	username   string
	passwordName string
	dataChannel channel.ISessionChannel
	first_ws_col uint32
	first_ws_row uint32
	flowLimit	int
	sendInterval	int
}

func StartPty(plugin *ShellPlugin)( err error) {
	finalCmd := "powershell.exe"
	if plugin.cmdContent != "" {
		finalCmd = plugin.cmdContent
	}
	log.GetLogger().Infoln("finalCmd ", finalCmd)
	exe_path,_ := pathutil.GetCurrentPath()
	winptyDllFilePath := exe_path + "Plugin/SessionManager/winpty.dll"
	var pty *winpty.WinPTY
	if plugin.username == "" {
		pty, err = winpty.Start(winptyDllFilePath, finalCmd, defaultConsoleCol, defaultConsoleRow, winpty.DEFAULT_WINPTY_FLAGS)
		if err != nil {
			log.GetLogger().Errorln("error in winpty.Start")
			return err
		}

	} else {
		log.GetLogger().Errorln("not support")
        return fmt.Errorf("not support")
	}

	plugin.pty = pty
	plugin.stdin = pty.StdIn
	plugin.stdout = pty.StdOut

	if plugin.first_ws_col != 0{
		plugin.SetSize(plugin.first_ws_col, plugin.first_ws_row)
	}
	return err
}

func (p *ShellPlugin) startPtyAsUser(user string, pass string, shellCmd string) (err error) {

	return nil
}

func (p *ShellPlugin) waitPid() () {

}

func (p *ShellPlugin) stop() (err error) {
	log.GetLogger().Info("Stopping winpty")
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

// InputStreamMessageHandler passes payload byte stream to shell stdin
func (p *ShellPlugin) InputStreamMessageHandler(streamDataMessage message.Message) error {
	if p.stdin == nil || p.stdout == nil {
		// This is to handle scenario when cli/console starts sending size data but pty has not been started yet
		// Since packets are rejected, cli/console will resend these packets until pty starts successfully in separate thread
		log.GetLogger().Tracef("Pty unavailable. Reject incoming message packet")
		return nil
	}

	switch streamDataMessage.MessageType {
	case message.InputStreamDataMessage:
		// log.Tracef("Output message received: %d", streamDataMessage.SequenceNumber)

		// deal with powershell nextline issue https://github.com/lzybkr/PSReadLine/issues/579
		payloadString := string(streamDataMessage.Payload)
		if strings.Contains(payloadString, "\r\n") {
			// From windows machine, do nothing
		} else if strings.Contains(payloadString, "\n") {
			// From linux machine, replace \n with \r
			num := strings.Index(payloadString, "\n")
			payloadString = strings.Replace(payloadString, "\n", "\r", num-1)
		}

		if _, err := p.stdin.Write([]byte(payloadString)); err != nil {
			log.GetLogger().Errorf("Unable to write to stdin, err: %v.", err)
			return err
		}
	case message.SetSizeDataMessage:
		var size SizeData
		if err := json.Unmarshal(streamDataMessage.Payload, &size); err != nil {
			log.GetLogger().Errorf("Invalid size message: %s", err)
			return err
		}
		log.GetLogger().Tracef("Resize data received: cols: %d, rows: %d", size.Cols, size.Rows)
		if err := p.SetSize(size.Cols, size.Rows); err != nil {
			log.GetLogger().Errorf("Unable to set pty size: %s", err)
			return err
		}
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

