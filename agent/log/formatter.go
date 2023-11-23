package log

import (
	"github.com/aliyun/aliyun_assist_client/thirdparty/sirupsen/logrus"
)

type CustomLogrusTextFormatter struct {
	LogrusTextFormatter logrus.TextFormatter

	CommonFields        logrus.Fields
}
