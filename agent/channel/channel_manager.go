package channel

import (
	"encoding/json"
	"errors"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/aliyun/aliyun_assist_client/agent/hybrid"

	"github.com/aliyun/aliyun_assist_client/agent/log"

	"github.com/aliyun/aliyun_assist_client/agent/taskengine"

	"github.com/aliyun/aliyun_assist_client/agent/util"
	"github.com/tidwall/gjson"
)

var _gshellChannel IChannel = nil

//manage all channels
type ChannelMgr struct {
	ActiveChannel   IChannel   //current used channel
	AllChannel      []IChannel //the first one is GshellChannel
	StopChanelEvent chan struct{}
	WaitCheckDone   sync.WaitGroup
	ChannelSetLock  sync.Mutex
}

//new
var G_ChannelMgr *ChannelMgr = &ChannelMgr{
	StopChanelEvent: make(chan struct{}),
}

// try to switch from non-gshell channel to gshell channel
func (m *ChannelMgr) checkChannelWorker() {
	m.ChannelSetLock.Lock()
	defer m.ChannelSetLock.Unlock()
	if m.AllChannel[0].IsWorking() {
		return
	}
	if !m.AllChannel[0].IsSupported() {
		return
	}
	if m.AllChannel[0].StartChannel() == nil {
		m.ActiveChannel.StopChannel()
		m.ActiveChannel = m.AllChannel[0]
	}
}

func (m *ChannelMgr) SelectAvailableChannel() error {
	m.ChannelSetLock.Lock()
	defer m.ChannelSetLock.Unlock()

	for _, item := range m.AllChannel {
		if item.IsWorking() {
			return nil
		}
	}
	for _, item := range m.AllChannel {
		if item.IsSupported() {
			if e := item.StartChannel(); e == nil {
				m.ActiveChannel = item
				return nil
			}
		}
	}
	return errors.New("No available channel")
}

func (m *ChannelMgr) Init(CallBack OnReceiveMsg) error {
	m.WaitCheckDone.Add(1)
	go func() {
		defer m.WaitCheckDone.Done()
		tick := time.NewTicker(time.Duration(1800) * time.Second)
		defer tick.Stop()
		for {
			select {
			case <-m.StopChanelEvent:
				return
			case <-tick.C:
				m.checkChannelWorker()
				m.SelectAvailableChannel()
			}
		}
	}()
	m.ChannelSetLock.Lock()
	defer m.ChannelSetLock.Unlock()
	m.AllChannel = append(m.AllChannel, _gshellChannel, NewWebsocketChannel(CallBack))
	if _gshellChannel.IsWorking() {
		m.ActiveChannel = _gshellChannel
		return nil
	}
	for _, item := range m.AllChannel {
		if item.IsSupported() {
			if e := item.StartChannel(); e == nil {
				m.ActiveChannel = item
				return nil
			}
		}
	}
	return errors.New("No available channel")
}

func (m *ChannelMgr) Uninit() {
	m.StopChanelEvent <- struct{}{}
	m.WaitCheckDone.Wait()
	m.ChannelSetLock.Lock()
	defer m.ChannelSetLock.Unlock()
	m.ActiveChannel.StopChannel()
}

func (m *ChannelMgr) GetCurrentChannelType() int {
	m.ChannelSetLock.Lock()
	defer m.ChannelSetLock.Unlock()
	return m.ActiveChannel.GetChannelType()
}

func InitChannelMgr(CallBack OnReceiveMsg) error {
	return G_ChannelMgr.Init(CallBack)
}

func StopChannelMgr() error {
	G_ChannelMgr.Uninit()
	return nil
}

func GetCurrentChannelType() int {
	return G_ChannelMgr.GetCurrentChannelType()
}

func TryStartGshellChannel() {
	_gshellChannel = NewGshellChannel(OnRecvMsg)
	if util.IsHybrid() == false {
		err := _gshellChannel.StartChannel()
		if err != nil {
			log.GetLogger().Infoln("TryStartGshellChannel failed ", err)
		} else {
			log.GetLogger().Infoln("TryStartGshellChannel ok ")
		}
	}
}

func StartChannelMgr() {
	InitChannelMgr(OnRecvMsg)
}

type GshellInvalid struct {
	Error struct {
		Class string `json:"class"`
		Desc  string `json:"desc"`
	} `json:"error"`
}

type GshellCheck struct {
	Execute   string `json:"execute"`
	Arguments struct {
		ID int64 `json:"id"`
	} `json:"arguments"`
}

type GshellCheckReply struct {
	Return int64 `json:"return"`
}

type GshellCmd struct {
	Execute   string `json:"execute"`
	Arguments struct {
		Cmd string `json:"cmd"`
	} `json:"arguments"`
}

type GshellCmdReply struct {
	Return struct {
		CmdOutput string `json:"cmd_output"`
		Result    int    `json:"result"`
	} `json:"return"`
}

type GshellShutdown struct {
	Execute   string `json:"execute"`
	Arguments struct {
		Mode string `json:"mode"`
	} `json:"arguments"`
}

func BuildInvalidRet(desc string) string {
	InvalidRet := GshellInvalid{}
	InvalidRet.Error.Class = "GenericError"
	InvalidRet.Error.Desc = desc
	retStr, _ := json.Marshal(InvalidRet)
	return string(retStr)
}

func onPowerdown() {
	shutdownCmd := "shutdown -h now"
	if runtime.GOOS == "windows" {
		shutdownCmd = "shutdown -f -s -t 0"
	}
	log.GetLogger().Infoln("powerdown......")
	util.ExeCmd(shutdownCmd)
}

func onReboot() {
	rebootCmd := "shutdown -r now"
	if runtime.GOOS == "windows" {
		rebootCmd = "shutdown -f -r -t 0"
	}
	log.GetLogger().Infoln("reboot......")
	util.ExeCmd(rebootCmd)
}

func OnRecvMsg(Msg string, ChannelType int) string {
	if ChannelType == ChannelWebsocketType {
		if Msg == "kick_vm" {
			go func() {
				taskengine.Fetch(true)
			}()
		} else if strings.Contains(Msg, "kick_vm agent deregister") {
			hybrid.UnRegister(true)
		}
		return ""
	}
	log.GetLogger().Infoln("kick msg:", Msg)
	if !gjson.Valid(Msg) {
		return BuildInvalidRet("invalid json1")
	}
	execute := gjson.Get(Msg, "execute").String()
	if execute == "guest-sync" {
		gshellCheck := GshellCheck{}
		err := json.Unmarshal([]byte(Msg), &gshellCheck)
		if err != nil {
			return BuildInvalidRet("invalid json: " + err.Error())
		}
		gshellCheckReply := GshellCheckReply{
			Return: gshellCheck.Arguments.ID,
		}
		retStr, _ := json.Marshal(gshellCheckReply)
		return string(retStr)
	} else if execute == "guest-command" {
		gshellCmd := GshellCmd{}
		err := json.Unmarshal([]byte(Msg), &gshellCmd)
		if err != nil {
			return BuildInvalidRet("invalid guest-command json: " + err.Error())
		}
		if gshellCmd.Arguments.Cmd == "kick_vm" {
			go func() {
				taskengine.Fetch(true)
			}()
		}
		gshellCmdReply := GshellCmdReply{}
		gshellCmdReply.Return.Result = 8
		gshellCmdReply.Return.CmdOutput = "execute kick_vm success"
		retStr, _ := json.Marshal(gshellCmdReply)
		return string(retStr)
	} else if execute == "guest-shutdown" {
		gshellShutdown := GshellShutdown{}
		err := json.Unmarshal([]byte(Msg), &gshellShutdown)
		if err != nil {
			return BuildInvalidRet("invalid guest-shutdown command: " + err.Error())
		}
		gshellCmdReply := GshellCmdReply{}
		gshellCmdReply.Return.Result = 8
		gshellCmdReply.Return.CmdOutput = "execute command success"
		retStr, _ := json.Marshal(gshellCmdReply)
		reboot := false
		if gshellShutdown.Arguments.Mode == "powerdown" {
		} else if gshellShutdown.Arguments.Mode == "reboot" {
			reboot = true
		} else {
			return BuildInvalidRet("invalid guest-shutdown command")
		}
		if reboot {
			onReboot()
		} else {
			onPowerdown()
		}
		return string(retStr)
	}
	return BuildInvalidRet("invalid command")
}
