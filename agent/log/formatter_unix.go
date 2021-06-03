// +build aix darwin dragonfly freebsd linux netbsd openbsd solaris

package log

import (
	"github.com/sirupsen/logrus"
)

func (f *CustomLogrusTextFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	return f.LogrusTextFormatter.Format(entry)
}
