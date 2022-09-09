package taskengine

import (
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"os"
	"os/user"
	"path"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/aliyun/aliyun_assist_client/agent/log"
	"github.com/aliyun/aliyun_assist_client/agent/metrics"
	"github.com/aliyun/aliyun_assist_client/agent/taskengine/models"
	"github.com/aliyun/aliyun_assist_client/agent/util"
)

const (
	ESuccess            = 0
	EFileCreateFail     = 1
	EChownError         = 2
	EChmodError         = 3
	ECreateDirFailed    = 4
	EInvalidFilePath    = 10
	EFileAlreadyExist   = 11
	EEmptyContent       = 12
	EInvalidContent     = 13
	EInvalidContentType = 14
	EInvalidFileType    = 15
	EInvalidSignature   = 16
	EInalidFileMode     = 17
	EInalidGID          = 18
	EInalidUID          = 19
)

var G_IsWindows bool = false
var G_IsFreebsd bool = false
var G_IsLinux bool = false

func init() {
	if runtime.GOOS == "windows" {
		G_IsWindows = true
	} else if runtime.GOOS == "linux" {
		G_IsLinux = true
	} else if runtime.GOOS == "freebsd" {
		G_IsFreebsd = true
	} else {

	}
}

func SendFiles(sendFileTasks []models.SendFileTaskInfo) {
	for _, s := range sendFileTasks {
		doSendFile(s)
	}
}

func SendFileFinished(sendFile models.SendFileTaskInfo, status int) {
	url := util.GetFinishOutputService()
	reportStatus := "Success"
	if status != ESuccess {
		reportStatus = "Failed"
	}
	param := fmt.Sprintf("?taskId=%s&status=%s&taskType=%s&errorcode=%d",
		sendFile.TaskID, reportStatus, "sendfile", status)
	url += param
	log.GetLogger().Printf("post = %s", url)
	if status != ESuccess {
		metrics.GetTaskFailedEvent(
			"taskid", sendFile.TaskID,
			"errormsg", param,
		).ReportEvent()
	}
	_, err := util.HttpPost(url, "", "text")
	if err != nil {
		log.GetLogger().Printf("HttpPost url %s error:%s ", url, err.Error())
	}
}

func SendFileInvalid(sendFile models.SendFileTaskInfo, status int) {
	url := util.GetInvalidTaskService()
	key := ""
	value := ""
	if status == EInvalidFilePath {
		key = "FileNameInvalid"
		value = sendFile.Name
	} else if status == EFileAlreadyExist {
		key = "FileExist"
		value = sendFile.Name
	} else if status == EEmptyContent {
		key = "EmptyFile"
	} else if status == EInvalidContent {
		key = "InvalidFileContent"
	} else if status == EInvalidSignature {
		key = "InvalidSignature"
		value = sendFile.Signature
	} else if status == EInalidFileMode {
		key = "InvalidFileMode"
		value = sendFile.Mode
	} else if status == EInalidGID {
		key = "FileGroupNotExist"
		value = sendFile.Group
	} else if status == EInalidUID {
		key = "FileOwnerNotExist"
		value = sendFile.Owner
	}
	metrics.GetTaskFailedEvent(
		"taskid", sendFile.TaskID,
		"errormsg", fmt.Sprintf("%s : %s", key, value),
	).ReportEvent()
	url = url + "?" + "taskId=" + sendFile.TaskID + "&taskType=sendfile&param=" + key + "&value=" + value
	log.GetLogger().Printf("post = %s", url)
	_, err := util.HttpPost(url, "", "text")
	if err != nil {
		log.GetLogger().Printf("HttpPost url %s error:%s ", url, err.Error())
	}
}

func doSendFile(task models.SendFileTaskInfo) {
	ret := sendFile(task)
	log.GetLogger().Println("sendFile ret: ", ret)
	if ret <= ECreateDirFailed {
		SendFileFinished(task, ret)
	} else {
		SendFileInvalid(task, ret)
	}
}

func sendFile(sendFile models.SendFileTaskInfo) int {
	if sendFile.Name == "" {
		return EInvalidFilePath
	}
	if sendFile.Content == "" {
		return EEmptyContent
	}
	fileDir := ""
	if sendFile.Destination == "" {
		if G_IsWindows {
			currentpath, _ := os.Executable()
			fileDir, _ = filepath.Abs(filepath.Dir(currentpath))

		} else {
			fileDir = "/root"
		}
	} else {
		fileDir = sendFile.Destination
	}

	if sendFile.Destination != "" {
		err := os.MkdirAll(sendFile.Destination, os.ModePerm)
		if err != nil {
			log.GetLogger().Errorln("MkdirAll error: ", err)
			return ECreateDirFailed
		}
	}
	if G_IsLinux || G_IsFreebsd {
		//文件下发时，如果root目录有一个test的文件，又创建了一个/root/test下的文件，则会报错。报错应通过invalid接口上报
		if util.IsFile(sendFile.Destination) {
			return EInvalidFilePath
		}
	}
	file_path := path.Join(fileDir, sendFile.Name)
	fileContent, err := base64.StdEncoding.DecodeString(sendFile.Content)
	if err != nil {
		log.GetLogger().Errorln("base64 decode error: ", err)
		return EInvalidContent
	}

	contentMd5 := util.ComputeStrMd5(sendFile.Content)

	if strings.ToLower(contentMd5) != strings.ToLower(sendFile.Signature) {
		return EInvalidSignature
	}
	fileMode := sendFile.Mode
	if len(fileMode) != 3 && len(fileMode) != 4 && len(fileMode) != 0 {
		return EInalidFileMode
	}
	if len(fileMode) == 0 {
		fileMode = "0644"
	}
	fMode, err := strconv.ParseInt(fileMode, 8, 32)
	if err != nil {
		return EInalidFileMode
	}
	ret := writeFile(file_path, fileContent, sendFile.Overwrite, os.FileMode(fMode))
	if ret != ESuccess {
		return ret
	}
	return changeFileOwner(file_path, sendFile.Owner, sendFile.Group)
}

func changeFileOwner(filePath string, User string, Group string) int {
	if G_IsWindows {
		return ESuccess
	}

	if User == "" && Group == "" {
		return ESuccess
	}

	if User == "root" && Group == "root" {
		return ESuccess
	}

	if User == "" {
		User = "root"
	}
	lu, err := user.Lookup(User)
	if err != nil {
		log.GetLogger().Printf("Lookup uid %s error:%s ", User, err.Error())
		return EInalidUID
	}
	uid, _ := strconv.Atoi(lu.Uid)
	if Group == "" {
		Group = "root"
	}
	lg, err := user.LookupGroup(Group)
	if err != nil {
		log.GetLogger().Printf("Lookup gid %s error:%s ", Group, err.Error())
		return EInalidGID
	}
	gid, _ := strconv.Atoi(lg.Gid)
	err = os.Chown(filePath, uid, gid)
	if err != nil {
		log.GetLogger().Printf("Chown file %s error:%s ", filePath, err.Error())
		return EChownError
	}
	return ESuccess
}

func writeFile(filePath string, data []byte, overWrite bool, fileMode os.FileMode) int {
	fileExist := util.FileExist(filePath)
	if fileExist && !overWrite {
		return EFileAlreadyExist
	}
	err := ioutil.WriteFile(filePath, data, 0644)
	if err != nil {
		log.GetLogger().Errorln("WriteFile: ", err)
		return EFileCreateFail
	}
	if G_IsLinux {
		err = os.Chmod(filePath, fileMode)
		if err != nil {
			log.GetLogger().Errorln(" Chmod faild", err)
			return EChmodError
		}
	}
	return ESuccess
}
