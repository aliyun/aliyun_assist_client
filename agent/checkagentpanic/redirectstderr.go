package checkagentpanic

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os"
	"time"

	"github.com/aliyun/aliyun_assist_client/agent/log"
)

const (
	checkAgentPanicTimeout = 10
)

var (
	panicInfoSizeLimit     = 50 * 1024

	lastPanicInfo      string
	lastPanicTimestamp time.Time

	errorFull error = fmt.Errorf("full")
	panicKeyWords [][]byte = [][]byte{[]byte("panic"), []byte("runtime"), []byte("fatal"), []byte("unexpected"), }
)

type limitedBuf struct {
	buf    []byte
	offset int
}

func newLimitBuf(size int) *limitedBuf {
	if size < 0 {
		size = 0
	}
	return &limitedBuf{
		buf: make([]byte, size),
	}
}

func (b *limitedBuf) Write(p []byte) (n int, err error) {
	if b.offset == len(b.buf) {
		return 0, errorFull
	}
	n = copy(b.buf[b.offset:], p)
	b.offset += n
	if n < len(p) {
		err = errorFull
	}
	return
}

func (b *limitedBuf) WriteChar(p byte) (err error) {
	if b.offset == len(b.buf) {
		err =  errorFull
		return
	}
	b.buf[b.offset] = p
	b.offset++
	return
}

func (b *limitedBuf) Content() string {
	return string(b.buf[:b.offset])
}

func searchPanicInfoFromFile(stderrFile string) string {
	logger := log.GetLogger().WithField("path", stderrFile)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(checkAgentPanicTimeout)*time.Second)
	defer cancel()
	stderrF, err := os.Open(stderrFile)
	if err != nil {
		logger.Error("Open stderr file failed: ", err)
		return ""
	}
	defer stderrF.Close()

	scanner := bufio.NewScanner(stderrF)
	// scanner.Split(bufio.ScanLines)
	scanner.Split(scanLines)
	var (
		ispanicInfo bool
		limitBuf    *limitedBuf = newLimitBuf(panicInfoSizeLimit)
	)
	var quit bool
	for !quit && scanner.Scan() {
		select {
		case <-ctx.Done():
			quit = true
		default:
		}

		line := scanner.Bytes()
		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}
		if ispanicInfo {
			if n, _ := limitBuf.Write(scanner.Bytes()); n == 0 {
				break
			}
		} else if checkIsPanicInfo(line) {
			ispanicInfo = true
			if n, _ := limitBuf.Write(scanner.Bytes()); n == 0 {
				break
			}
		}
	}
	return limitBuf.Content()
}

func scanLines(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}
	if i := bytes.IndexByte(data, '\n'); i >= 0 {
		return i + 1, data[0:i+1], nil
	}
	if atEOF {
		return len(data), data, nil
	}
	return 0, nil, nil
}

func checkIsPanicInfo(line []byte) bool {
	for _, k := range panicKeyWords {
		if bytes.Contains(line, k) {
			return true
		}
	}
	return false
}