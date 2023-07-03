//go:build aix || darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris
// +build aix darwin dragonfly freebsd linux netbsd openbsd solaris

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
	return f.LogrusTextFormatter.Format(entry)
}
