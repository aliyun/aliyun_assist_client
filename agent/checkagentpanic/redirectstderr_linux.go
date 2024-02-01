//go:build linux

package checkagentpanic

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/aliyun/aliyun_assist_client/agent/util"
	"github.com/aliyun/aliyun_assist_client/common/executil"
	"github.com/aliyun/aliyun_assist_client/thirdparty/sirupsen/logrus"
	"github.com/tidwall/gjson"
)

const (
	journalctlRealTimeKey   = "__REALTIME_TIMESTAMP"
	journalctlMessageKey    = "MESSAGE"
	findCheckPointCmdFormat = "journalctl -u aliyun --no-pager --quiet --output=json --since \"%s\" --reverse | grep \"%s\" | grep \"aliyun-service\" -m 1"
	getjournalLogCmdFormat  = "journalctl -u aliyun --no-pager --quiet --output=json --since \"%s\" | grep \"aliyun-service\""
	journalctlTimeLimit     = -time.Hour * 24 * 7
	timeFormat              = "2006-01-02 15:04:05"
)

func searchPanicInfoFromJournalctl(logger logrus.FieldLogger) (time.Time, string) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(checkAgentPanicTimeout)*time.Second)
	defer cancel()
	sinceTime := time.Now().Add(journalctlTimeLimit)
	sinceTimeStr := sinceTime.Format(timeFormat)
	cmd := fmt.Sprintf(findCheckPointCmdFormat, sinceTimeStr, checkPoint)
	err, stdout, stderr := util.ExeCmdWithContext(ctx, cmd)
	if err != nil {
		logger.Errorf("Can not find last check point from journal. cmd[%s], err[%v], stdout[%s], stderr[%s] ",
			cmd, err, stdout, stderr)
	} else {
		stdout = strings.TrimSpace(stdout)
		if stdout == "" {
			logger.Errorf("Can not find last check point from journal. cmd[%s], err[%v], stdout[%s], stderr[%s] ",
				cmd, err, stdout, stderr)
		} else {
			_, timestampstr, _ := parseJournalLine(logger, []byte(stdout))
			if timestampstr != "" {
				sinceTimeStr = timestampstr
			}
		}
	}
	if ctx.Err() != nil {
		logger.Error("Find agent panic log from journal timeout.")
		return time.Time{}, ""
	}

	cmd = fmt.Sprintf(getjournalLogCmdFormat, sinceTimeStr)
	r, err := execCmd(ctx, cmd)
	if err != nil {
		logger.Error("Find agent panic log from journal failed. cmd[%s], err[%v]", cmd, err)
		return time.Time{}, ""
	}
	scanner := bufio.NewScanner(r)
	scanner.Split(bufio.ScanLines)
	var (
		ispanicInfo bool
		limitBuf    *limitedBuf = newLimitBuf(panicInfoSizeLimit)
		panicTime   time.Time
	)
	for scanner.Scan() {
		line := scanner.Bytes()
		timestamp, timestampstr, message := parseJournalLine(logger, line)
		if timestampstr == "" || message == "" {
			continue
		} else {
			if ispanicInfo {
				limitBuf.Write([]byte(message))
				if limitBuf.WriteChar('\n') != nil {
					break
				}
			} else if strings.HasPrefix(message, "panic:") {
				panicTime = timestamp
				ispanicInfo = true
				limitBuf.Write([]byte(message))
				limitBuf.WriteChar('\n')
			}
		}
	}

	return panicTime, limitBuf.Content()
}

func parseJournalLine(logger logrus.FieldLogger, line []byte) (timestamp time.Time, timestampstr, message string) {
	if !gjson.ValidBytes(line) {
		logger.Errorf("Invalid json: %s", string(line))
		return
	}
	realTimeFiled := gjson.GetBytes(line, journalctlRealTimeKey)
	if !realTimeFiled.Exists() {
		logger.Errorf("Not found %s in `%s`", journalctlRealTimeKey, string(line))
		return
	} else {
		if timestampmill, err := strconv.ParseInt(realTimeFiled.String(), 10, 64); err != nil {
			logger.Errorf("parse %s from `%s` failed: %v", journalctlRealTimeKey, string(line), err)
			return
		} else {
			timestamp = time.UnixMicro(timestampmill)
			timestampstr = timestamp.Format(timeFormat)
		}
	}
	messageField := gjson.GetBytes(line, journalctlMessageKey)
	if !messageField.Exists() {
		logger.Errorf("not found %s in `%s`", journalctlMessageKey, string(line))
		return
	} else {
		message = messageField.String()
	}
	return
}

// execCmd start command 'cmd' with a context and return command.Stdout
func execCmd(ctx context.Context, cmd string) (io.ReadCloser, error) {
	command := executil.CommandWithContext(ctx, "sh", "-c", cmd)
	pr, err := command.StdoutPipe()
	if err != nil {
		return nil, err
	}
	if err := command.Start(); err != nil {
		return nil, err
	}
	// The command.Wait will close pr. There is a race condition between
	// command.Wait() goroutine and pr's reading goroutine, it is not a big
	// problem for now.
	go command.Wait()
	return pr, nil
}
