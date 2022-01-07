package channel

import (
	"encoding/json"
	"reflect"
	"testing"

	"bou.ke/monkey"
	"github.com/aliyun/aliyun_assist_client/agent/kickvmhandle"
	"github.com/aliyun/aliyun_assist_client/agent/update"
	"github.com/aliyun/aliyun_assist_client/agent/util"
)

func TestOnRecvMsg(t *testing.T) {
	type args struct {
		Msg         string
		ChannelType int
	}
	theArgs := args{
		Msg: "kick_vm",
		ChannelType: ChannelGshellType,
	}
	tests := []struct {
		name string
		subname string
		args args
		want string
	}{
		{
			name: "ws",
			subname: "CriticalActionRunning",
			args: theArgs,
		},
		{
			name: "ws",
			args: theArgs,
		},
		{
			name: "guest-sync",
			args: theArgs,
		},
		{
			name: "guest-sync",
			subname: "CriticalActionRunning",
			args: theArgs,
		},
		{
			name: "guest-command",
			subname: "CriticalActionRunning",
			args: theArgs,
		},
		{
			name: "guest-command",
			subname: "kick_vm",
			args: theArgs,
		},
		{
			name: "guest-command",
			subname: "valid agent",
			args: theArgs,
		},
		{
			name: "guest-command",
			subname: "invalid agent",
			args: theArgs,
		},
		{
			name: "guest-shutdown",
			subname: "reboot",
			args: theArgs,
		},
		{
			name: "guest-shutdown",
			subname: "powerdown",
			args: theArgs,
		},
		{
			name: "guest-shutdown",
			subname: "unknown",
			args: theArgs,
		},
	}
	guard := monkey.Patch(util.ExeCmd, func(string) (error, string, string) { return nil, "", ""} )
	defer guard.Unpatch()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.name == "ws" {
				tt.args.ChannelType = ChannelWebsocketType
			}

			if tt.subname == "CriticalActionRunning" {
				guard := monkey.Patch(update.IsCriticalActionRunning, func() bool { return true })
				defer guard.Unpatch()
			} else {
				guard := monkey.Patch(update.IsCriticalActionRunning, func() bool { return false })
				defer guard.Unpatch()
			}

			if tt.name == "guest-sync" {
				msg := GshellCheck{
					Execute: "guest-sync",
				}
				msg.Arguments.ID = 10000
				content, _ := json.Marshal(&msg)
				tt.args.Msg = string(content)
			}
			
			if tt.name == "guest-command" {
				msg := GshellCmd{
					Execute: "guest-command",
				}
				if tt.subname == "kick_vm" {
					msg.Arguments.Cmd = "kick_vm"
				} else if tt.subname == "valid agent" {
					msg.Arguments.Cmd = "valid agent params params"
					var a *kickvmhandle.AgentHandle
					guard := monkey.PatchInstanceMethod(reflect.TypeOf(a), "CheckAction", func(*kickvmhandle.AgentHandle) bool { return true })
					defer guard.Unpatch()
				} else if tt.subname == "invalid agent" {
					msg.Arguments.Cmd = "invalid agent params params"
					var a *kickvmhandle.AgentHandle
					guard := monkey.PatchInstanceMethod(reflect.TypeOf(a), "CheckAction", func(*kickvmhandle.AgentHandle) bool { return false })
					defer guard.Unpatch()
				}
				content, _ := json.Marshal(&msg)
				tt.args.Msg = string(content)
			}

			if tt.name == "guest-shutdown" {
				msg := GshellShutdown{
					Execute: "guest-shutdown",
				}
				msg.Arguments.Mode = tt.subname
				content, _ := json.Marshal(&msg)
				tt.args.Msg = string(content)
			}
			OnRecvMsg(tt.args.Msg, tt.args.ChannelType)
			// if got := OnRecvMsg(tt.args.Msg, tt.args.ChannelType); got != tt.want {
			// 	t.Errorf("OnRecvMsg() = %v, want %v", got, tt.want)
			// }
		})
	}
}
