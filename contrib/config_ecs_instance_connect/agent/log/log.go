package log

import (
	"log/syslog"
	"fmt"
	"os"
)


var sysLog *syslog.Writer

func init() {
	var err error
	sysLog, err = syslog.Dial("", "", syslog.LOG_ERR, "config_ecs_instance_connect")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func GetLogger() *syslog.Writer {
	return sysLog
}

func Info(strs ...interface{}) {
	sysLog.Info(fmt.Sprint(strs))
}