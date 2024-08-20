package log

import (
	"os"
	"path/filepath"
	"time"

	rotatelogs "github.com/lestrrat-go/file-rotatelogs"
	log "github.com/sirupsen/logrus"
	"github.com/pkg/errors"
)

var Log *log.Logger
var defaultLevel log.Level = log.InfoLevel

type Fields = log.Fields

// DefaultCommonFields returns preset fields for each log message. All default
// common fields MUST be prefixed by an underscore to avoid conflict with fields
// added later.
func DefaultCommonFields() log.Fields {
	return log.Fields{
		"_pid": os.Getpid(),
	}
}

func InitLog(filename string, logpath string, ignoreRotationError bool) {
	if logpath != "" {
		logpath = filepath.Join(logpath, "log")
	} else {
		path, _ := os.Executable()
		logpath = filepath.Join(filepath.Dir(path), "log")
	}
	os.MkdirAll(logpath, os.ModePerm)


	writer, err := rotatelogs.New(
		filepath.Join(logpath, filename+".%Y%m%d"),
		rotatelogs.WithMaxAge(time.Duration(24*30)*time.Hour),    //最长保留30天
		rotatelogs.WithRotationTime(time.Duration(24)*time.Hour), //每天进行一次日志切割
		rotatelogs.WithLinkName(filepath.Join(logpath, filename)),         // 为日志文件创建一个名字不变的链接
	)
	if err != nil {
		log.Errorf("config local file system logger error. %+v", errors.WithStack(err))
	}

	Log = log.New()
	Log.SetFormatter(&CustomLogrusTextFormatter{
		CommonFields: DefaultCommonFields(),
	})
	Log.SetOutput(writer)
	Log.SetLevel(defaultLevel)
}

func GetLogger() *log.Logger {
	if Log == nil {
		InitLog("aliyun_ecs_session_log", "", false)
	}
	return Log
}
