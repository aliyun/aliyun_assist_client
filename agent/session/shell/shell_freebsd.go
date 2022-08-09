package shell

import (
	"github.com/aliyun/aliyun_assist_client/agent/session/channel"
	"github.com/aliyun/aliyun_assist_client/agent/session/message"
	"os"
)

type ShellPlugin struct {
	id        string
	stdin       *os.File
	stdout      *os.File
	cmdContent        string
	username   string
	passwordName string
	dataChannel channel.ISessionChannel
	flowLimit	int
	sendInterval	int
}

const (
	termEnvVariable       = "TERM=xterm-256color"
	langEnvVariable       = "LANG=C.UTF-8"
	langEnvVariableKey    = "LANG"
	homeEnvVariable       = "HOME=/home/"
	default_runas_user    = "ecs-assist-user"
)

func StartPty(plugin *ShellPlugin)( err error) {
	return nil
}

func (p *ShellPlugin) stop() (err error) {
	return nil
}

func (p *ShellPlugin) SetSize(ws_col, ws_row uint32) (err error) {
	return nil
}

func (p *ShellPlugin) InputStreamMessageHandler(streamDataMessage message.Message) error {
	return nil
}

func (p *ShellPlugin) waitPid() () {

}
