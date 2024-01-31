package taskengine

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/aliyun/aliyun_assist_client/agent/log"
	"github.com/aliyun/aliyun_assist_client/agent/metrics"
	"github.com/aliyun/aliyun_assist_client/agent/session/channel"
	"github.com/aliyun/aliyun_assist_client/agent/session/port"
	"github.com/aliyun/aliyun_assist_client/agent/session/shell"
	"github.com/aliyun/aliyun_assist_client/agent/taskengine/models"
	"github.com/aliyun/aliyun_assist_client/agent/util"
)

type SessionTask struct {
	taskId       string
	sessionId    string
	websocketUrl string

	cmdContent   string
	username     string
	passwordName string
	targetHost   string
	portNumber   string
	flowLimit    int

	sessionChannel *channel.SessionChannel
	shellPlugin    *shell.ShellPlugin
	portPlugin     *port.PortPlugin
	cancelFlag     util.CancelFlag
}

func NewSessionTask(sessionId string, websocketUrl string, taskId string,
	cmdContent string, username string, passwordName string, targetHost string,
	portNumber string, flowLimit int) *SessionTask {
	task := &SessionTask{
		sessionId:    sessionId,
		taskId:       taskId,
		websocketUrl: websocketUrl,

		cmdContent:   cmdContent,
		passwordName: passwordName,
		username:     username,
		targetHost:   targetHost,
		portNumber:   portNumber,
		flowLimit:    flowLimit,

		cancelFlag: util.NewChanneledCancelFlag(),
	}
	return task
}

func ReportSessionResult(taskID string, status string) {
	url := util.GetSessionStatusService()
	reportStatus := "Failed"
	if status == shell.Ok || status == shell.Notified || status == shell.Timeout {
		reportStatus = "Success"
	}
	param := fmt.Sprintf("?channelId=%s&status=%s&errorcode=%s",
		taskID, reportStatus, status)
	url += param
	log.GetLogger().Printf("post = %s", url)

	_, err := util.HttpPost(url, "", "text")
	if err != nil {
		metrics.GetTaskFailedEvent(
			"errormsg", fmt.Sprintf("report session result err: %s", err.Error()),
			"url", url,
			"taskid", taskID,
		).ReportEvent()
		log.GetLogger().Printf("HttpPost url %s error:%s ", url, err.Error())
	}
}

func (sessionTask *SessionTask) isPortForwardTask() bool {
	return sessionTask.portNumber != "" 
}

func (sessionTask *SessionTask) runTask() (string, error) {
	ret := GetSessionFactory().ContainsTask(sessionTask.sessionId)
	if ret == true {
		log.GetLogger().Errorln("NewSessionChannel failed")
		return shell.Session_id_duplicate, errors.New("NewSessionChannel failed")
	}
	if sessionTask.isPortForwardTask() {
		port_num, _ := strconv.Atoi(sessionTask.portNumber)
		sessionTask.portPlugin = port.NewPortPlugin(sessionTask.sessionId, sessionTask.targetHost, port_num, sessionTask.flowLimit)
	} else {
		sessionTask.shellPlugin = shell.NewShellPlugin(sessionTask.sessionId, sessionTask.cmdContent, sessionTask.username, sessionTask.passwordName, sessionTask.flowLimit)
	}
	GetSessionFactory().AddSessionTask(sessionTask)

	host := util.GetServerHost()
	if host == "" {
		return shell.Init_channel_failed, errors.New("No available host")
	}

	websocketUrl := "wss://" + host + "/luban/session/backend?channelId=" + sessionTask.sessionId
	log.GetLogger().Infoln("url: ", websocketUrl)
	var err error
	var session_channel *channel.SessionChannel
	if sessionTask.isPortForwardTask() {
		session_channel, err = channel.NewSessionChannel(websocketUrl, sessionTask.sessionId, sessionTask.portPlugin.InputStreamMessageHandler, sessionTask.cancelFlag)

	} else {
		session_channel, err = channel.NewSessionChannel(websocketUrl, sessionTask.sessionId, sessionTask.shellPlugin.InputStreamMessageHandler, sessionTask.cancelFlag)
		sessionTask.sessionChannel = session_channel
	}

	if err != nil {
		log.GetLogger().Errorln("NewSessionChannel failed", err)
		return shell.Init_channel_failed, fmt.Errorf("NewSessionChannel failed: %v", err)
	}
	sessionTask.sessionChannel = session_channel

	err = session_channel.Open()
	if err != nil {
		log.GetLogger().Errorln("NewSessionChannel failed", err)
		return shell.Open_channel_failed, fmt.Errorf("NewSessionChannel failed: %v", err)
	}

	done := make(chan int, 1)
	error_code := shell.Ok

	go func() {
		time.Sleep(1 * time.Second)
		if sessionTask.isPortForwardTask() {
			log.GetLogger().Infoln("run portPlugin")
			error_code, err = sessionTask.portPlugin.Execute(session_channel, sessionTask.cancelFlag)
		} else {
			log.GetLogger().Infoln("run shellPlugin")
			error_code, err = sessionTask.shellPlugin.Execute(session_channel, sessionTask.cancelFlag)
		}

		done <- 1
	}()

	select {
	case <-done:
		log.GetLogger().Println("shell end", sessionTask.sessionId)
	case <-time.After(time.Duration(3600*3) * time.Second):
		log.GetLogger().Println("shell timeout", sessionTask.sessionId)
		error_code = shell.Timeout
	}

	return error_code, err
}

func DoSessionTask(tasks []models.SessionTaskInfo) {
	go func() {
		for _, s := range tasks {
			session := NewSessionTask(s.SessionId,
				s.WebsocketUrl,
				s.SessionId,
				s.CmdContent,
				s.Username,
				s.Password,
				s.TargetHost,
				s.PortNumber,
				s.FlowLimit)
			session.RunTask(s.SessionId)
		}
	}()
}

func (sessionTask *SessionTask) RunTask(taskid string) error {
	log.GetLogger().Infoln("run task", taskid, sessionTask.sessionId)
	code, err := sessionTask.runTask()
	ReportSessionResult(taskid, code)
	if sessionTask.sessionChannel != nil {
		sessionTask.sessionChannel.Close()
	}
	GetSessionFactory().RemoveTask(sessionTask.sessionId)
	if err != nil {
		metrics.GetTaskFailedEvent(
			"errormsg", err.Error(),
			"taskid", sessionTask.taskId,
			"sessionid", sessionTask.sessionId,
			"code", code,
			"wsURL", sessionTask.websocketUrl,
		).ReportEvent()
		metrics.GetSessionFailedEvent(
			"sessionId", sessionTask.sessionId,
			"errormsg", err.Error(),
			"taskid", sessionTask.taskId,
			"code", code,
			"wsURL", sessionTask.websocketUrl,
		).ReportEvent()
	}
	return err
}

func (sessionTask *SessionTask) StopTask() error {
	log.GetLogger().Infoln("stop task", sessionTask.taskId)
	if sessionTask.shellPlugin != nil || sessionTask.portPlugin != nil {
		sessionTask.cancelFlag.Set(util.Completed)
	} else {
		log.GetLogger().Errorln("sesison plugin is invalid")
	}

	return nil
}
