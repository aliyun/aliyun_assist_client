package log

import (
	"os"
	"path/filepath"
	"time"

	rotatelogs "github.com/lestrrat-go/file-rotatelogs"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

var Log *log.Logger

func InitLog(filename string) {
	path, _ := os.Executable()
	dir, _ := filepath.Abs(filepath.Dir(path))
	writer, err := rotatelogs.New(
		dir+"/log/"+filename+".%Y%m%d",
		rotatelogs.WithMaxAge(time.Duration(24*30)*time.Hour),    //最长保留30天
		rotatelogs.WithRotationTime(time.Duration(24)*time.Hour), //每天进行一次日志切割
	)
	if err != nil {
		log.Errorf("config local file system logger error. %+v", errors.WithStack(err))
	}

	Log = log.New()
	Log.SetOutput(writer)
}

func GetLogger() *log.Logger {
	if Log == nil {
		InitLog("aliyun_assist_test")
	}
	return Log
}
