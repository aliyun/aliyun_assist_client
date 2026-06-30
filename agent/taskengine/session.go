package taskengine

import (
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/aliyun/aliyun_assist_client/agent/log"
	"github.com/aliyun/aliyun_assist_client/agent/metrics"
	"github.com/aliyun/aliyun_assist_client/agent/session/channel"
	"github.com/aliyun/aliyun_assist_client/agent/session/port"
	sessionresult "github.com/aliyun/aliyun_assist_client/agent/session/sessionresult"
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

	encryptionOptions models.EncryptionOptions

	sessionChannel *channel.SessionChannel
	shellPlugin    *shell.ShellPlugin
	portPlugin     *port.PortPlugin
	cancelFlag     util.CancelFlag
}

const (
	sessionTimeoutSecond = 3 * 3600
)

func NewSessionTask(sessionId string, websocketUrl string, taskId string,
	cmdContent string, username string, passwordName string, targetHost string,
	portNumber string, flowLimit int, encryptionOptions models.EncryptionOptions) *SessionTask {
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

		encryptionOptions: encryptionOptions,

		cancelFlag: util.NewChanneledCancelFlag(),
	}
	return task
}

func ReportSessionResult(taskID string, sessionRes *sessionresult.SessionResult) {
	fullurl := util.GetSessionStatusService()
	var reportStatus string
	// sessionRes should not be nil but we still check it in case of unexpect case.
	if sessionRes == nil {
		sessionRes = sessionresult.NewOk("")
	}
	if sessionRes.OK {
		reportStatus = "Success"
	} else {
		reportStatus = "Failed"
	}

	param := url.Values{}
	param.Add("channelId", taskID)
	param.Add("status", reportStatus)
	param.Add("errorcode", sessionRes.Code)
	param.Add("errorinfo", sessionRes.Info)
	fullurl += "?" + param.Encode()
	log.GetLogger().Printf("post = %s", fullurl)

	_, err := util.HttpPost(fullurl, "", "text")
	if err != nil {
		metrics.GetTaskFailedEvent(
			"errormsg", fmt.Sprintf("report session result err: %s", err.Error()),
			"url", fullurl,
			"taskid", taskID,
		).ReportEvent()
		log.GetLogger().Printf("HttpPost url %s error:%s ", fullurl, err.Error())
	}
}

func (sessionTask *SessionTask) isPortForwardTask() bool {
	return sessionTask.portNumber != ""
}

func (sessionTask *SessionTask) runTask() *sessionresult.SessionResult {
	if GetSessionFactory().ContainsTask(sessionTask.sessionId) {
		log.GetLogger().Errorln("NewSessionChannel failed")
		return sessionresult.NewSessionIdDuplicateError(sessionTask.sessionId)
	}
	if sessionTask.isPortForwardTask() {
		port_num, _ := strconv.Atoi(sessionTask.portNumber)
		sessionTask.portPlugin = port.NewPortPlugin(sessionTask.sessionId, sessionTask.targetHost, port_num, sessionTask.flowLimit)
	} else {
		sessionTask.shellPlugin = shell.NewShellPlugin(sessionTask.sessionId, sessionTask.cmdContent, sessionTask.username, sessionTask.passwordName, sessionTask.flowLimit, sessionTask.encryptionOptions)
	}
	GetSessionFactory().AddSessionTask(sessionTask)

	host := util.GetServerHost()
	if host == "" {
		return sessionresult.NewServerDomainUnavailableError()
	}

	websocketUrl := "wss://" + host + "/luban/session/backend?channelId=" + sessionTask.sessionId
	log.GetLogger().Infoln("url: ", websocketUrl)
	var err error
	var session_channel *channel.SessionChannel
	if sessionTask.isPortForwardTask() {
		session_channel = channel.NewSessionChannel(websocketUrl, sessionTask.sessionId, sessionTask.portPlugin.InputStreamMessageHandler, sessionTask.cancelFlag)
	} else {
		session_channel = channel.NewSessionChannel(websocketUrl, sessionTask.sessionId, sessionTask.shellPlugin.InputStreamMessageHandler, sessionTask.cancelFlag)
	}
	sessionTask.sessionChannel = session_channel

	err = session_channel.Open()
	if err != nil {
		log.GetLogger().Errorln("NewSessionChannel failed", err)
		return sessionresult.NewOpenChannelFailedError(err)
	}

	done := make(chan int, 1)
	var sessionRes *sessionresult.SessionResult

	go func() {
		time.Sleep(1 * time.Second)
		if sessionTask.isPortForwardTask() {
			log.GetLogger().Infoln("run portPlugin")
			sessionRes = sessionTask.portPlugin.Execute(session_channel, sessionTask.cancelFlag)
		} else {
			log.GetLogger().Infoln("run shellPlugin")
			sessionRes = sessionTask.shellPlugin.Execute(session_channel, sessionTask.cancelFlag)
		}

		done <- 1
	}()

	select {
	case <-done:
		log.GetLogger().Println("shell end", sessionTask.sessionId)
	case <-time.After(time.Duration(sessionTimeoutSecond) * time.Second):
		log.GetLogger().Println("shell timeout", sessionTask.sessionId)
		sessionRes = sessionresult.NewSessionTimeoutError(sessionTimeoutSecond)
	}

	// For shell session task, if error occurred try to send error message to client.
	// sessionRes should not be nil but we still check it in case of unexpect case.
	if !sessionTask.isPortForwardTask() &&
		sessionRes != nil &&
		sessionRes.Code != sessionresult.Ok &&
		session_channel != nil {
		session_channel.SendStreamDataMessage([]byte("The session has terminated. " + sessionRes.Error() + "\n"))
		time.Sleep(time.Second)
	}

	return sessionRes
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
				s.FlowLimit,
				s.EncryptionOptions)
			session.RunTask(s.SessionId)
		}
	}()
}

func (sessionTask *SessionTask) RunTask(taskid string) {
	log.GetLogger().Infoln("run task", taskid, sessionTask.sessionId)
	sessionRes := sessionTask.runTask()
	ReportSessionResult(taskid, sessionRes)
	if sessionTask.sessionChannel != nil {
		sessionTask.sessionChannel.Close()
	}
	GetSessionFactory().RemoveTask(sessionTask.sessionId)
	if !sessionRes.OK {
		metrics.GetTaskFailedEvent(
			"errormsg", sessionRes.Info,
			"taskid", sessionTask.taskId,
			"sessionid", sessionTask.sessionId,
			"code", sessionRes.Code,
			"wsURL", sessionTask.websocketUrl,
		).ReportEvent()
		metrics.GetSessionFailedEvent(
			"sessionId", sessionTask.sessionId,
			"errormsg", sessionRes.Info,
			"taskid", sessionTask.taskId,
			"code", sessionRes.Code,
			"wsURL", sessionTask.websocketUrl,
		).ReportEvent()
	}
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
