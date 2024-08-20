//go:build aix || darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris
// +build aix darwin dragonfly freebsd linux netbsd openbsd solaris

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
	return f.LogrusTextFormatter.Format(entry)
}
