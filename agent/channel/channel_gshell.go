package channel

import (
	"encoding/json"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/aliyun/aliyun_assist_client/agent/log"
	"github.com/aliyun/aliyun_assist_client/agent/util"
)

type GshellChannel struct {
	*Channel
	hGshell         *os.File
	StopChanelEvent chan struct{}
	WaitCheckDone   sync.WaitGroup
	lock            sync.Mutex
}

type gshellStatus struct {
	gshellSupport string `json:"gshellSupport"`
}

func (c *GshellChannel) IsSupported() bool {
	if util.IsHybrid() {
		return false
	}
	c.lock.Lock()
	if !c.Working {
		if c.startChannelUnsafe() != nil {
			c.lock.Unlock()
			return false
		}
	}
	c.lock.Unlock()
	url := util.GetGshellCheckService()
	resp, err := util.HttpPost(url, "", "text")
	if err != nil {
		log.GetLogger().Errorln("HttpPost ", url, "error! ", err)
		return false
	}
	log.GetLogger().Infoln("HttpPost ", url, "OK! resp: ", resp)

	var gstatus gshellStatus
	if err := json.Unmarshal([]byte(resp), &gstatus); err != nil {
		return false
	}
	if gstatus.gshellSupport == "true" {
		return true
	}
	return false
}

// func (c *GshellChannel) canOpenGshell() bool {
// 	gShellSupport := false
// 	c.lock.Lock()
// 	defer c.lock.Unlock()
// 	gshellPath := "/dev/virtio-ports/org.qemu.guest_agent.0"
// 	if runtime.GOOS == "windows" {
// 		gshellPath = "\\\\.\\Global\\org.qemu.guest_agent.0"
// 	}
// 	if f, e := os.Open(gshellPath); e == nil {
// 		gShellSupport = true
// 		f.Close()
// 	}
// 	log.GetLogger().Infoln("open gshell status:", gshellPath, gShellSupport)
// 	return gShellSupport
// }

func (c *GshellChannel) StartChannel() error {
	c.lock.Lock()
	defer c.lock.Unlock()
	if c.Working {
		return nil
	}
	return c.startChannelUnsafe()
}

func (c *GshellChannel) startChannelUnsafe() error {
	gshellPath := "/dev/virtio-ports/org.qemu.guest_agent.0"
	if runtime.GOOS == "windows" {
		gshellPath = "\\\\.\\Global\\org.qemu.guest_agent.0"
	}
	h, e := os.OpenFile(gshellPath, os.O_RDWR, 0666)
	if e != nil {
		log.GetLogger().Errorln("open gshell failed:", gshellPath, "error:", e)
		return e
	}
	log.GetLogger().Infoln("open gshell ok:", gshellPath)
	c.hGshell = h
	c.WaitCheckDone.Add(1)
	go func() {
		defer c.WaitCheckDone.Done()
		tick := time.NewTicker(time.Duration(200) * time.Millisecond)
		defer tick.Stop()
		buf := make([]byte, 2048)
		for {
			select {
			case <-c.StopChanelEvent:
				return
			case <-tick.C:
				n, err := c.hGshell.Read(buf)
				if err == nil && n > 0 {
					retStr := c.CallBack(string(buf[:n]), ChannelGshellType)
					if len(retStr) > 0 {
						log.GetLogger().Infoln("write:", retStr)
						_, err = c.hGshell.Write([]byte(retStr + "\n"))
						if err != nil {
							log.GetLogger().Errorln("write error:", err)
							go c.SwitchChannel()
						}
					}
				}
			}
		}
	}()
	c.Working = true
	return nil
}

func (c *GshellChannel) StopChannel() error {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.StopChanelEvent <- struct{}{}
	c.WaitCheckDone.Wait()
	c.hGshell.Close()
	c.Working = false
	return nil
}

func (c *GshellChannel) SwitchChannel() error {
	c.StopChannel()
	time.Sleep(time.Duration(3) * time.Second)
	return G_ChannelMgr.SelectAvailableChannel()
}

func NewGshellChannel(CallBack OnReceiveMsg) IChannel {
	return &GshellChannel{
		Channel: &Channel{
			CallBack:    CallBack,
			ChannelType: ChannelGshellType,
			Working:     false,
		},
		StopChanelEvent: make(chan struct{}),
	}
}
