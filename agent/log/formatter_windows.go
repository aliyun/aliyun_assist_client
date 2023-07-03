//go:build windows
// +build windows

package log

import (
	"github.com/aliyun/aliyun_assist_client/thirdparty/sirupsen/logrus"
)

func (f *CustomLogrusTextFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	if f.Fileds != nil {
		for k, v := range f.Fileds {
			entry.Data[k] = v
		}
	}
	logrusTextFormatted, err := f.LogrusTextFormatter.Format(entry)
	if logrusTextFormatted != nil && len(logrusTextFormatted) > 0 {
		logrusTextFormatted[len(logrusTextFormatted)-1] = '\r'
		logrusTextFormatted = append(logrusTextFormatted, '\n')
	}

	return logrusTextFormatted, err
}
