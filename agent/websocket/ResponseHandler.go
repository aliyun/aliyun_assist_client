package websocket

/*
import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/aliyun/aliyun_assist_client/agent/util"
	"github.com/aliyun/aliyun_assist_client/agent/util/httputil"

	"github.com/tidwall/gjson"
)

var G_ResponseHandlerLock sync.Mutex

func ResponseHandler(responseData string, fromHttp bool) error {
	G_ResponseHandlerLock.Lock()
	defer G_ResponseHandlerLock.Unlock()
	MyInfo.Println("response: ", responseData, "fromHttp :", fromHttp)
	if responseData == "ok" {
		return nil
	}
	if !gjson.Valid(responseData) {
		return errors.New("invalid json")
	}
	actionType := gjson.Get(responseData, "actionType")
	actionId := gjson.Get(responseData, "actionId")
	reply := &replyToServer{
		ActionType: actionType.String(),
		ActionID:   actionId.String(),
	}
	switch actionType.String() {
	case "getLog":
		reply.Result = getLogHandler(gjson.Get(responseData, "logPath").String())
	case "installAssit":
		reply.Result = installAssist(gjson.Get(responseData, "InstallInfo.url").String(), gjson.Get(responseData, "InstallInfo.md5").String(), gjson.Get(responseData, "InstallInfo.version").String())
	case "updateDeamon":
		reply.Result = updateAssistDeamon(gjson.Get(responseData, "updateInfo.url").String(), gjson.Get(responseData, "updateInfo.md5").String(), gjson.Get(responseData, "updateInfo.version").String())
	default:
		return errors.New("invalid actionType")
	}
	jsonBytes, err := json.Marshal(*reply)
	if err != nil {
		return err
	}
	if fromHttp {
		_, err = httputil.HttpPost(util.GetDeamonUrl(), string(jsonBytes), "")
	} else {
		err = ReplyMsgToWebsocketServer(string(jsonBytes))
	}
	if err != nil {
		MyError.Println("reply error:", err.Error())
	}
	return err
}

type replyToServer struct {
	ActionType string `json:"actionType"`
	ActionID   string `json:"actionId"`
	Result     string `json:"Result"`
}

func getLogHandler(logPath string) string {
	if logPath == "" {
		return "error! logpath is null"
	}
	if !strings.HasPrefix(logPath, "C:\\ProgramData\\aliyun\\") &&
		!strings.HasPrefix(logPath, "/usr/local/share/aliyun") {
		return "error! invalid logpath"
	}
	f, err := ioutil.ReadFile(logPath)
	if err != nil {
		return "error! " + err.Error()
	}
	return string(f)
}

func updateAssistDeamon(url string, md5 string, version string) string {
	if G_AssistVersion == version {
		return "ok"
	}
	DestExe := "/usr/local/share/AssistDeamon"
	if G_IsWindows {
		DestExe = "C:\\ProgramData\\aliyun\\AssistDeamon.exe"
	}
	destAssistDir := "/usr/local/share/assist-deamon"
	if G_IsWindows {
		destAssistDir = "C:\\ProgramData\\aliyun\\assist-deamon"
	}
	os.MkdirAll(destAssistDir, os.ModePerm)
	if err := os.MkdirAll(filepath.Dir(DestExe), os.ModePerm); err != nil {
		return err.Error()
	}
	err := util.HttpDownlod(url, DestExe)
	if err != nil {
		return err.Error()
	}
	tmpMd5, err := util.ComputeMd5(DestExe)
	if err != nil {
		return err.Error()
	}
	if tmpMd5 != md5 {
		return "error: file md5 is not valid"
	}
	if !G_IsWindows {
		err = ioutil.WriteFile("/usr/local/share/UpdateAssistDeamon.sh", []byte(updateDeamonLinux), 0644)
	} else {
		err = ioutil.WriteFile("C:\\ProgramData\\aliyun\\UpdateAssistDeamon.bat", []byte(updateDeamonWin), 0644)
	}
	if err != nil {
		return err.Error()
	}
	if !G_IsWindows {
		util.ExeCmd("chmod +x /usr/local/share/UpdateAssistDeamon.sh")
	}
	if G_IsWindows {
		err, _ = util.ExeCmdNoWait("C:\\ProgramData\\aliyun\\UpdateAssistDeamon.bat")
	} else {
		err, _ = util.ExeCmdNoWait("/usr/local/share/UpdateAssistDeamon.sh")
	}
	if err != nil {
		return err.Error()
	}
	return "ok"
}

func installAssist(url string, md5 string, version string) string {
	if G_AssistVersion == version {
		return "ok"
	}
	if G_IsWindows {
		return installAssistWin(url, md5, version)
	} else {
		return installAssistLinux(url, md5, version)
	}
}

func installAssistLinux(url string, md5 string, version string) string {
	bRpm := util.HasCmdInLinux("rpm")
	DestExe := "/root/tmp_assist"
	err := util.HttpDownlod(url, DestExe)
	if err != nil {
		return err.Error()
	}
	tmpMd5, err := util.ComputeMd5(DestExe)
	if err != nil {
		return err.Error()
	}
	if tmpMd5 != md5 {
		return "error: file md5 is not valid"
	}
	if bRpm {
		util.ExeCmd("rpm --force -ivh  /root/tmp_assist")
	} else {
		util.ExeCmd("dpkg -i /root/tmp_assist")
	}
	if G_AssistVersion == version {
		return "ok"
	}
	return "Install assist failed"
}

func installAssistWin(url string, md5 string, version string) string {
	cpath, _ := os.Getwd()
	tmpZip := filepath.Join(cpath, "tmp_assist.zip")
	err := util.HttpDownlod(url, tmpZip)
	if err != nil {
		return err.Error()
	}
	defer os.Remove("tmp_assist.zip")
	tmpMd5, err := util.ComputeMd5(tmpZip)
	if err != nil {
		return err.Error()
	}
	if tmpMd5 != md5 {
		return "error: file md5 is not valid"
	}
	tmpZipDir := filepath.Join(cpath, "tmp_assist")
	tmpZipDir = filepath.Join(tmpZipDir, version)
	err = util.Unzip(tmpZip, tmpZipDir)
	if err != nil {
		return err.Error()
	}
	defer os.RemoveAll("tmp_assist")
	destAssistDir := "/usr/local/share/aliyun-assist/" + version
	if G_IsWindows {
		destAssistDir = "C:\\ProgramData\\aliyun\\assist\\" + version
	}
	os.MkdirAll(destAssistDir, os.ModePerm)
	copyScript := "mv " + tmpZipDir + " " + destAssistDir
	if G_IsWindows {
		copyScript = "echo d | xcopy /e /q /y " + tmpZipDir + " " + destAssistDir
	}
	err, out, stderr := util.ExeCmd(copyScript)
	if err != nil {
		MyInfo.Println(copyScript, "\n", out)
		fmt.Println(err.Error(), "\n", stderr)
		return err.Error()
	}

	installScript := filepath.Join(destAssistDir, "update_install")
	if G_IsWindows {
		installScript = filepath.Join(destAssistDir, "install.bat")
	}
	err, out, _ = util.ExeCmd(installScript)
	if err != nil {
		MyInfo.Println(installScript, "\n", out)
		return err.Error()
	}
	if G_AssistVersion == version {
		return "ok"
	}
	return "Install assist failed"
}

func getErrorString(err error, stderr string) string {
	errStr := ""
	if err != nil {
		errStr += err.Error()
	}
	errStr += stderr
	return "error:" + errStr
}

const updateDeamonWin = `
C:\ProgramData\aliyun\assist-deamon\AssistDeamon.exe stop
C:\ProgramData\aliyun\assist-deamon\AssistDeamon.exe remove
copy /y C:\ProgramData\aliyun\AssistDeamon.exe  C:\ProgramData\aliyun\assist-deamon\AssistDeamon.exe
del /q C:\ProgramData\aliyun\AssistDeamon.exe
C:\ProgramData\aliyun\assist-deamon\AssistDeamon.exe install
C:\ProgramData\aliyun\assist-deamon\AssistDeamon.exe start
`

const updateDeamonLinux = `
/usr/local/share/assist-deamon/AssistDeamon stop
/usr/local/share/assist-deamon/AssistDeamon remove
cp -f /usr/local/share/AssistDeamon /usr/local/share/assist-deamon/AssistDeamon
chmod +x /usr/local/share/assist-deamon/AssistDeamon
/usr/local/share/assist-deamon/AssistDeamon install
/usr/local/share/assist-deamon/AssistDeamon start
rm -rf /usr/local/share/AssistDeamon
`

 */
