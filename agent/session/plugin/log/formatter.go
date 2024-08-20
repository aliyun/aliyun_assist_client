package log

import (
	"github.com/sirupsen/logrus"
)

type CustomLogrusTextFormatter struct {
	LogrusTextFormatter logrus.TextFormatter

	CommonFields        logrus.Fields
}
