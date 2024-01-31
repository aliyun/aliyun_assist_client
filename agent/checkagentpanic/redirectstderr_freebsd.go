//go:build freebsd

package checkagentpanic

import (
	"time"
	"github.com/aliyun/aliyun_assist_client/thirdparty/sirupsen/logrus"
)

func getStderrLogPath() (string, error) {
	return "/var/log/aliyun-service.log", nil
}

func searchPanicInfoFromJournalctl(logger logrus.FieldLogger) (time.Time, string) {
	return time.Time{}, ""
}
