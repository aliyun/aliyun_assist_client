//go:build linux

package checkagentpanic

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"encoding/json"
	"time"
	"strconv"

	"github.com/aliyun/aliyun_assist_client/agent/util"
	"github.com/aliyun/aliyun_assist_client/thirdparty/sirupsen/logrus"
)

const (
	journalctlRealTimeKey   = "__REALTIME_TIMESTAMP"
	journalctlMessageKey    = "MESSAGE"
	findCheckPointCmdFormat = "journalctl -u aliyun --no-pager --quiet --output=json --reverse | grep \"%s\" | grep \"aliyun-service\" -m 1"
	getjournalLogCmdFormat  = "journalctl -u aliyun --no-pager --quiet --output=json --since \"%s\" | grep \"aliyun-service\""
)

func getStderrLogPath() (string, error) {
	selfPath, err := os.Executable()
	if err != nil {
		return "", err
	}
	selfName := filepath.Base(selfPath)
	return fmt.Sprintf("/var/log/%s.err", selfName), nil
}

func searchPanicInfoFromJournalctl(logger logrus.FieldLogger) (panicTime string, panicInfo []string) {
	cmd := fmt.Sprintf(findCheckPointCmdFormat, checkPoint)
	var stdout string
	err, stdout, _ := util.ExeCmd(cmd)
	if err != nil {
		logger.Errorf("find last check point from journal failed, cmd[%s], err[%v] ", cmd, err)
		return
	}
	stdout = strings.TrimSpace(stdout)
	if stdout == "" {
		logger.Info("not found check point")
		return
	}
	timestampstr, _ := parseJournalLine(logger, stdout)
	cmd = fmt.Sprintf(getjournalLogCmdFormat, timestampstr)
	err, stdout, _ = util.ExeCmd(cmd)
	if err != nil {
		logger.Errorf("find log from journal since last check poing failed, cmd[%s], err[%v]", cmd, err)
		return
	}
	lines := strings.Split(stdout, "\n")
	var ispanicInfo bool
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		timestamp, message := parseJournalLine(logger, line)
		if timestamp == "" || message == "" {
			continue
		} else {
			if ispanicInfo {
				panicInfo = append(panicInfo, message)
			} else if strings.HasPrefix(message, "panic:") {
				panicTime = timestamp
				ispanicInfo = true
				panicInfo = append(panicInfo, message)
			}
		}
	}
	return
}

func parseJournalLine(logger logrus.FieldLogger, line string) (timestampstr, message string) {
	record := map[string]string{}
	if err := json.Unmarshal([]byte(line), &record); err != nil {
		logger.Errorf("json unmarshal `%s` failed: %v", line, err)
		return
	}
	if value, ok := record[journalctlRealTimeKey]; !ok {
		logger.Errorf("not found %s in `%s`", journalctlRealTimeKey, line)
		return
	} else {
		if timestampmill, err := strconv.ParseInt(value, 10, 64); err != nil {
			logger.Errorf("parse %s from `%s` failed: %v", journalctlRealTimeKey, line, err)
			return
		} else {
			timestamp := time.UnixMicro(timestampmill)
			timestampstr = timestamp.Format("2006-01-02 15:04:05")
		}
	}

	if value, ok := record[journalctlMessageKey]; !ok {
		logger.Errorf("not found %s in `%s`", journalctlMessageKey, line)
		return
	} else {
		message = value
	}
	return
}
