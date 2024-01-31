package channel

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/tidwall/gjson"

	"github.com/aliyun/aliyun_assist_client/agent/clientreport"
	"github.com/aliyun/aliyun_assist_client/agent/hybrid"
	"github.com/aliyun/aliyun_assist_client/agent/kickvmhandle"
	"github.com/aliyun/aliyun_assist_client/agent/log"
	"github.com/aliyun/aliyun_assist_client/agent/metrics"
	"github.com/aliyun/aliyun_assist_client/agent/taskengine"
	"github.com/aliyun/aliyun_assist_client/agent/update"
	"github.com/aliyun/aliyun_assist_client/agent/util/powerutil"
	"github.com/aliyun/aliyun_assist_client/common/apiserver"
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
func (m *ChannelMgr) checkChannelWorker() bool {
	log.GetLogger().Infoln("checkChannelWorker")
	m.ChannelSetLock.Lock()
	defer m.ChannelSetLock.Unlock()
	if m.AllChannel[0].IsSupported() && m.AllChannel[0].IsWorking() {
		if m.AllChannel[1].IsWorking() {
			m.AllChannel[1].StopChannel()
		}
		m.ActiveChannel = m.AllChannel[0]
		return true
	}
	return false
}

func (m *ChannelMgr) SelectAvailableChannel(currentChannel int) error {
	log.GetLogger().Infoln("SelectAvailableChannel")
	m.ChannelSetLock.Lock()
	defer m.ChannelSetLock.Unlock()

	for _, item := range m.AllChannel {
		if currentChannel == item.GetChannelType() {
			continue
		}
		if item.IsSupported() && item.IsWorking() {
			return nil
		}
	}
	for _, item := range m.AllChannel {
		if currentChannel == item.GetChannelType() {
			continue
		}
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
				if m.checkChannelWorker() == false {
					err := m.SelectAvailableChannel(ChannelNone)
					var report clientreport.ClientReport
					if err == nil {
						report = clientreport.ClientReport{
							ReportType: "switch_channel_in_timer",
							Info:       fmt.Sprintf("success: Current channel is %d", m.GetCurrentChannelType()),
						}
					} else {
						report = clientreport.ClientReport{
							ReportType: "switch_channel_in_timer",
							Info:       fmt.Sprintf("fail:" + err.Error()),
						}
					}
					metrics.GetChannelSwitchEvent(
						"type", ChannelTypeStr(m.GetCurrentChannelType()),
						"reportType", report.ReportType,
						"info", report.Info,
					).ReportEvent()
					clientreport.SendReport(report)
				}
			}
		}
	}()
	m.ChannelSetLock.Lock()
	defer m.ChannelSetLock.Unlock()
	m.AllChannel = append(m.AllChannel, _gshellChannel, NewWebsocketChannel(CallBack))
	for _, item := range m.AllChannel {
		if item.IsSupported() {
			if e := item.StartChannel(); e == nil {
				m.ActiveChannel = item
				return nil
			} else {
				fmt.Println(e.Error())
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
	if m.ActiveChannel == nil {
		return ChannelNone
	}
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
	if apiserver.IsHybrid() == false {
		err := _gshellChannel.StartChannel()
		if err != nil {
			log.GetLogger().Infoln("TryStartGshellChannel failed ", err)
		} else {
			log.GetLogger().Infoln("TryStartGshellChannel ok ")
		}
	}
}

func StartChannelMgr() {
	if err := InitChannelMgr(OnRecvMsg); err != nil {
		metrics.GetChannelFailEvent(
			metrics.EVENT_SUBCATEGORY_CHANNEL_MGR,
			"type", "channelmgr",
			"errormsg", err.Error(),
		).ReportEvent()
	}
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
		// netcheck field would not presented when no diagnostic result available
		Netcheck *NetcheckReply `json:"netcheck,omitempty"`
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

func OnRecvMsg(Msg string, ChannelType int) string {
	log.GetLogger().Infoln("kick msg:", Msg)

	// legacy code for websocket kick data proc.
	if ChannelType == ChannelWebsocketType {
		if update.IsCriticalActionRunning() {
			return "reject:" + Msg
		}
		if Msg == "kick_vm" {
			go func() {
				taskengine.Fetch(true, "", taskengine.NormalTaskType)			}()
			return "accept:" + Msg
		} else if strings.Contains(Msg, "kick_vm agent deregister") {
			hybrid.UnRegister(true)
		}

		handle := kickvmhandle.ParseOption(Msg)
		valid_cmd := false
		if handle != nil {
			if handle.CheckAction() == true {
				valid_cmd = true
				go func() {
					handle.DoAction()
				}()
			}
		}
		if valid_cmd == false {
			return "unknow:" + Msg
		} else {
			return "accept:" + Msg
		}

	}

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
		if update.IsCriticalActionRunning() {
			gshellCmdReply := GshellCmdReply{}
			gshellCmdReply.Return.Result = 7
			gshellCmdReply.Return.CmdOutput = "agent is busy"
			retStr, _ := json.Marshal(gshellCmdReply)
			return string(retStr)
		}
		if gshellCmd.Arguments.Cmd == "kick_vm" {
			go func() {
				taskengine.Fetch(true, "", taskengine.NormalTaskType)			}()
			gshellCmdReply := GshellCmdReply{}
			gshellCmdReply.Return.Result = 8
			gshellCmdReply.Return.CmdOutput = "execute kick_vm success"
			gshellCmdReply.Return.Netcheck = LastNetcheckReply()
			retStr, _ := json.Marshal(gshellCmdReply)
			return string(retStr)
		} else {
			handle := kickvmhandle.ParseOption(gshellCmd.Arguments.Cmd)
			valid_cmd := false
			if handle != nil {
				if handle.CheckAction() == true {
					valid_cmd = true
					go func() {
						handle.DoAction()
					}()
				}
			}
			if valid_cmd == false {
				gshellCmdReply := GshellCmdReply{}
				gshellCmdReply.Return.Result = 6
				gshellCmdReply.Return.CmdOutput = "invalid command"
				retStr, _ := json.Marshal(gshellCmdReply)
				return string(retStr)
			} else {
				gshellCmdReply := GshellCmdReply{}
				gshellCmdReply.Return.Result = 8
				gshellCmdReply.Return.CmdOutput = "execute kick_vm success"
				gshellCmdReply.Return.Netcheck = LastNetcheckReply()
				retStr, _ := json.Marshal(gshellCmdReply)
				return string(retStr)
			}
		}
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
		if gshellShutdown.Arguments.Mode == powerutil.PowerdownMode {
		} else if gshellShutdown.Arguments.Mode == powerutil.RebootMode {
			reboot = true
		} else {
			return BuildInvalidRet("invalid guest-shutdown command")
		}
		powerutil.Shutdown(reboot)
		return string(retStr)
	}
	return BuildInvalidRet("invalid command")
}
