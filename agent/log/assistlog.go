package log

import (
	"os"
	"path/filepath"
	"time"

	log "github.com/aliyun/aliyun_assist_client/thirdparty/sirupsen/logrus"
	rotatelogs "github.com/lestrrat-go/file-rotatelogs"
	"github.com/pkg/errors"
)

var Log *log.Logger
var defaultLevel log.Level = log.InfoLevel

func InitLog(filename string, logpath string) {
	logdir := ""
	if logpath == "" {
		path, _ := os.Executable()
		logdir, _ = filepath.Abs(filepath.Dir(path))
	} else {
		logdir = logpath
	}

	writer, err := rotatelogs.New(
		logdir+"/log/"+filename+".%Y%m%d",
		rotatelogs.WithMaxAge(time.Duration(24*30)*time.Hour),    //最长保留30天
		rotatelogs.WithRotationTime(time.Duration(24)*time.Hour), //每天进行一次日志切割
		rotatelogs.WithLinkName(logdir+"/log/"+filename),         // 为日志文件创建一个名字不变的链接
	)
	if err != nil {
		log.Errorf("config local file system logger error. %+v", errors.WithStack(err))
	}

	Log = log.New()
	Log.SetFormatter(&CustomLogrusTextFormatter{})
	Log.SetOutput(writer)
	Log.SetLevel(defaultLevel)
}

func GetLogger() *log.Logger {
	if Log == nil {
		InitLog("aliyun_assist_test", "")
	}
	return Log
}
