package taskengine

import (
	"errors"
	"fmt"
	"github.com/aliyun/aliyun_assist_client/agent/log"
	"github.com/aliyun/aliyun_assist_client/agent/session/channel"
	"github.com/aliyun/aliyun_assist_client/agent/session/shell"
	"github.com/aliyun/aliyun_assist_client/agent/util"
	"strconv"
	"time"
)

type SessionTask struct{
	taskId       string
	sessionId    string
	websocketUrl string
	cmdContent       string
	username   string
	passwordName string
	sessionChannel      *channel.SessionChannel
	shellPlugin         *shell.ShellPlugin
	cancelFlag         util.CancelFlag
}


func NewSessionTask(sessionId string,
	                websocketUrl string,
	                taskId string, cmdContent string, username string, passwordName string) *SessionTask{
	task := &SessionTask{
		sessionId:sessionId,
		taskId:taskId,
		websocketUrl:websocketUrl,
		cmdContent:cmdContent,
		passwordName:passwordName,
		username:username,
		cancelFlag: util.NewChanneledCancelFlag(),
	}
	return task
}

func ReportSessionResult(taskID string, status string) {
	url := util.GetSessionStatusService()
	reportStatus := "Success"
	if status != shell.Ok {
		reportStatus = "Failed"
	}
	param := fmt.Sprintf("?channelId=%s&status=%s&errorcode=%s",
		taskID, reportStatus, status)
	url += param
	log.GetLogger().Printf("post = %s", url)

	_, err := util.HttpPost(url, "", "text")
	if err != nil {
		log.GetLogger().Printf("HttpPost url %s error:%s ", url, err.Error())
	}
}

func (sessionTask *SessionTask) runTask() (string, error){
	ret := GetSessionFactory().ContainsTask(sessionTask.sessionId)
	if ret == true {
		log.GetLogger().Errorln("NewSessionChannel failed")
		return  shell.Session_id_duplicate, errors.New("NewSessionChannel failed")
	}
	shellPlugin := shell.NewShellPlugin(sessionTask.sessionId, sessionTask.cmdContent, sessionTask.username, sessionTask.passwordName)
	GetSessionFactory().AddSessionTask(sessionTask)

	host := util.GetServerHost()
	if host == "" {
		return  shell.Init_channel_failed, errors.New("No available host")
	}

	websocketUrl :=  "wss://" + host + "/luban/session/backend?channelId=" + sessionTask.sessionId
	log.GetLogger().Infoln("url: ", websocketUrl)
	session_channel, err := channel.NewSessionChannel(websocketUrl, sessionTask.sessionId, shellPlugin.InputStreamMessageHandler, sessionTask.cancelFlag)
	sessionTask.sessionChannel = session_channel
	if err != nil {
		log.GetLogger().Errorln("NewSessionChannel failed", err)
		return  shell.Init_channel_failed, errors.New("NewSessionChannel failed")
	}

	err = session_channel.Open()
	if err != nil {
		log.GetLogger().Errorln("NewSessionChannel failed", err)
		return  shell.Open_channel_failed, errors.New("NewSessionChannel failed")
	}

	done := make(chan int, 1)
	error_code := shell.Ok

	go func() {
		time.Sleep(1*time.Second)
		log.GetLogger().Infoln("run shellPlugin")
		error_code = shellPlugin.Execute(session_channel, sessionTask.cancelFlag)
		done <- 1
	}()

	select {
		case <-done:
			log.GetLogger().Println("shell end", sessionTask.sessionId)
		case <-time.After(time.Duration(3600*3) * time.Second):
			log.GetLogger().Println("shell timeout", sessionTask.sessionId)
			error_code = shell.Time_out
	}

	return error_code,nil
}

func DoSessionTask(tasks [] SessionTaskInfo) {
	go func() {
		for _, s := range tasks {
			session := NewSessionTask(s.SessionId,
				s.WebsocketUrl,
				s.SessionId,
				s.CmdContent,
				s.Username,
				s.Password)
			session.RunTask(s.SessionId)
		}
	}()
}

// for debug purpose.
var debug_session_id int = 0
func DoDebugSessionTask() {
	go func() {
		debug_session_id ++
		session_id := strconv.Itoa(debug_session_id)
		host := "x.x.x.x:8090"
		url := "ws://" + host + "/luban/start_session_from_vm"
		session := NewSessionTask(session_id,
			url,
			"i-xxxxxxxxxxxx",
			"",
			"",
			"")
		session.RunTask(session.taskId)
	}()
}


func (sessionTask *SessionTask) RunTask(taskid string) error{
	log.GetLogger().Infoln("run task", taskid, sessionTask.sessionId)
	code,err := sessionTask.runTask()
	ReportSessionResult(taskid, code)
	sessionTask.sessionChannel.Close()
	GetSessionFactory().RemoveTask(sessionTask.sessionId)
	return err
}

func (sessionTask *SessionTask) StopTask() error{
	log.GetLogger().Infoln("stop task", sessionTask.taskId)
	if sessionTask.shellPlugin != nil {
		sessionTask.cancelFlag.Set(util.Canceled)
	}

	return nil
}