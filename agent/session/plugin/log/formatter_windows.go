//go:build windows
// +build windows

package log

import (
	"github.com/sirupsen/logrus"
)

func (f *CustomLogrusTextFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	if f.CommonFields != nil {
		for k, v := range f.CommonFields {
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
