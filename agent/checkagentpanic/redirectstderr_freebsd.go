//go:build freebsd

package checkagentpanic

import (
	"github.com/aliyun/aliyun_assist_client/thirdparty/sirupsen/logrus"
)

func getStderrLogPath() (string, error) {
	return "/var/log/aliyun-service.log", nil
}

func searchPanicInfoFromJournalctl(logger logrus.FieldLogger) (panicTime string, panicInfo []string) {
	return "", nil
}
