package checkospanic

import (
	"bufio"
	"bytes"
	"compress/flate"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/aliyun/aliyun_assist_client/agent/log"
	"github.com/aliyun/aliyun_assist_client/agent/metrics"
	"github.com/aliyun/aliyun_assist_client/common/fileutil"
	"github.com/aliyun/aliyun_assist_client/agent/util/timetool"
	"github.com/aliyun/aliyun_assist_client/thirdparty/sirupsen/logrus"
)

const (
	kdumpPath            = "/var/crash"
	kdumpConfigPath      = "/etc/kdump.conf"
	vmcoreDmesgFile      = "vmcore-dmesg.txt"

	maxLinesBeforePanicInfo  = 200
	maxLinesAfterPanicInfo   = 300
)

var (
	// 127.0.0.1-2023-07-05-20:51:21 or <hostname>-2023-07-05-20:51:21
	vmcorePathRegex      = regexp.MustCompile(`^(?:[\w.-]+)-(\d{4}-\d{2}-\d{2}-\d{2}:\d{2}:\d{2})$`)

	rePanicInfoMatch      = regexp.MustCompile(`(?:\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}\.\d{3}[+-]\d{4})?(?:\[\s*\d+\.\d+\])?\s*([^\n]+)`)
	reRIPMatch            = regexp.MustCompile(`RIP.*?([\w-.]+\+0x\w+)/0x`)
	reLinuxCallTraceMatch = regexp.MustCompile(`(\[[\d. ]+]\s+)*(<\w+> )*(\[(<\w+>)?] )?(\? )?([\w-.]+\+0x\w+)/0x`) // which comes from dmesg

	panicMsgs = []string{
		"SysRq : Crash",
		"SysRq : Trigger a crash",
		"SysRq : Netdump",
		"general protection fault: ",
		"double fault: ",
		"divide error: ",
		"stack segment: ",
		"Oops: ",
		"Kernel BUG at",
		"kernel BUG at",
		"BUG: unable to handle page fault for address",
		"BUG: unable to handle kernel ",
		"Unable to handle kernel paging request",
		"Unable to handle kernel NULL pointer dereference",
		"Kernel panic: ",
		"Kernel panic - ",
		"[Hardware Error]: ",
		"Bad mode in ",
	}
)

func ReportLastOsPanic() {
	logger := log.GetLogger().WithField("Phase", "ReportLastOsPanic")
	vmcoreDmesgPath, vmcoreDir, latestTime := FindLocalVmcoreDmesg(logger)
	if vmcoreDmesgPath == "" {
		logger.Info("there is no vmcore file need report")
		return
	}
	if time.Since(latestTime) > time.Hour*24 {
		logger.Info("the latest vmcore file is 24 hours ago, ignore it")
		return
	}
	if !fileutil.CheckFileIsExist(vmcoreDmesgPath) {
		logger.Error("vmcore dmesg file not exist", vmcoreDmesgPath)
		return
	}
	logger = logger.WithField("file", vmcoreDmesgPath)
	var kernelPanicInfo, rip, callTrace string
	rip, callTrace, kernelPanicInfo, rawContent, err := ParseVmcore(logger, vmcoreDmesgPath)
	if err != nil {
		logger.Error("parse vmcore dmesg file failed: ", err)
		return
	}
	compressedContent, err := compressFlate(rawContent)
	if err != nil {
		tip := fmt.Sprint("compress raw content failed: ", err)
		logger.Error(tip)
	}
	_, _, timeZone := timetool.NowWithTimezoneName()
	metrics.GetLinuxGuestOSPanicEvent(
		"rip", rip,
		"callTrace", callTrace,
		"kernelPanicInfo", kernelPanicInfo,
		"crashTime", latestTime.Format("2006-01-02 15:04:05"),
		"crashTimeUTC", latestTime.UTC().Format("2006-01-02 15:04:05"),
		"timeZone", timeZone,
		"vmcoreDir", vmcoreDir,
		"rawContent", compressedContent,
	).ReportEvent()
	logger.Info("the latest vmcore file has reported")
}

// ParseVmcore parse fileds `Call Trace` `RIP` `Kernel Panic` from vmcore-dmesg.txt
func ParseVmcore(logger logrus.FieldLogger, vmcoreDmesgPath string) (rip, callTrace, panicInfo, rawContent string, err error) {
	var (
		callTraceLines []string

		rawContentBeforPanicInfo []string
		rawContentAfterPanicInfo []string
	)
	var vmcoreFile *os.File
	vmcoreFile, err = os.Open(vmcoreDmesgPath)
	if err != nil {
		logger.Error("open vmcore dmesg file failed: ", err)
		return
	}
	defer vmcoreFile.Close()

	scanner := bufio.NewScanner(vmcoreFile)
	scanner.Split(bufio.ScanLines)
	panicInfo, rawContentBeforPanicInfo = parsePanicInfo(scanner, maxLinesBeforePanicInfo)
	if panicInfo == "" {
		err = fmt.Errorf("panic info not found")
		return
	}
	// found panicInfo, go on to find rip and call trace
	var inCallTrace bool
	var done int = 2 // two items need find: callTrace and rip
	for done != 0 && scanner.Scan() {
		line := scanner.Text()
		if len(rawContentAfterPanicInfo) < maxLinesAfterPanicInfo {
			rawContentAfterPanicInfo = append(rawContentAfterPanicInfo, line)
		}
		if strings.Contains(strings.ToLower(line), "call trace:") {
			if inCallTrace {
				break
			}
			inCallTrace = true
			continue
		}

		if inCallTrace {
			if reLinuxCallTraceMatch.MatchString(line) {
				callTraceLines = append(callTraceLines, line)
			} else {
				done--
			}
		}

		if rip != "" && reRIPMatch.MatchString(line) {
			rip = reRIPMatch.FindStringSubmatch(line)[1]
			done--
		}
	}
	for len(rawContentAfterPanicInfo) < maxLinesAfterPanicInfo && scanner.Scan() {
		rawContentAfterPanicInfo = append(rawContentAfterPanicInfo, scanner.Text())
	}
	callTrace = strings.Join(callTraceLines, "\n")
	rawContent = strings.Join(rawContentBeforPanicInfo, "\n")
	if len(rawContentAfterPanicInfo) > 0 {
		rawContent = rawContent + "\n" + strings.Join(rawContentAfterPanicInfo, "\n")
	}
	return
}

// FindLocalVmcoreDmesg find latest directory which stores the vmcore-dmesg.txt
func FindLocalVmcoreDmesg(logger logrus.FieldLogger) (vmcoreDmesgPath, latestDir string, latestTime time.Time) {
	kdumpDirTemp := kdumpPath
	// read /etc/kdump.conf to get vmcore directory, default is /var/crash
	if fileutil.CheckFileIsExist(kdumpConfigPath) {
		content, err := os.ReadFile(kdumpConfigPath)
		if err == nil {
			lines := strings.Split(string(content), "\n")
			for _, line := range lines {
				if strings.HasPrefix(line, "path ") {
					fields := strings.Fields(line)
					if len(fields) == 2 {
						kdumpDirTemp = strings.TrimSpace(fields[1])
					}
				}
			}
		}
	}
	if !fileutil.CheckFileIsExist(kdumpDirTemp) {
		logger.WithField("path", kdumpDirTemp).Warn("kdump directory not exist")
		return
	}
	entries, err := os.ReadDir(kdumpDirTemp)
	if err != nil {
		logger.WithFields(logrus.Fields{
			"path": kdumpDirTemp,
			"err":  err,
		}).Error("read kdump directory failed")
		return
	}
	for _, entry := range entries {
		if entry.IsDir() && vmcorePathRegex.MatchString(entry.Name()) {
			vmcoreDir := entry.Name()
			items := vmcorePathRegex.FindStringSubmatch(vmcoreDir)
			if len(items) != 2 {
				logger.Error("unknown vmcore directory name fromation:", vmcoreDir)
			} else {
				vmcoreTime, err := time.ParseInLocation("2006-01-02-15:04:05", items[1], time.Local)
				if err != nil {
					logger.WithFields(logrus.Fields{
						"name": vmcoreDir,
						"err":  err,
					}).Error("parse time from vmcore directory name failed")
				} else {
					if latestDir == "" || vmcoreTime.Sub(latestTime) > 0 {
						latestDir = vmcoreDir
						latestTime = vmcoreTime
					}
				}
			}
		}
	}
	if latestDir == "" {
		return
	}
	vmcoreDmesgPath = filepath.Join(kdumpDirTemp, latestDir, vmcoreDmesgFile)
	return
}

// compress input by flate algorithm, and encode by base64
func compressFlate(input string) (string, error) {
	if len(input) == 0 {
		return "", nil
	}
	buf := new(bytes.Buffer)
	flateWriter, err := flate.NewWriter(buf, flate.BestCompression)
	if err != nil {
		return "", err
	}
	defer flateWriter.Close()
	flateWriter.Write([]byte(input))
	flateWriter.Flush()
	return base64.StdEncoding.EncodeToString(buf.Bytes()), nil
}

// Find the first panicInfo log and return the contents of the n logs before it
func parsePanicInfo(scanner *bufio.Scanner, n int) (panicInfo string, content []string) {
	ringbuf := make(chan string, n)
	var found bool
	for !found && scanner.Scan() {
		line := scanner.Text()
		select {
		case ringbuf<-line:
		default:
			<-ringbuf
			ringbuf<-line
		}
		for _, panicMsg := range panicMsgs {
			if strings.Contains(line, panicMsg) {
				panicInfo = rePanicInfoMatch.FindStringSubmatch(line)[1] //获取关键信息
				found = true
				break
			}
		}
	}
	content = make([]string, 0, n)
	var done bool
	for !done {
		select {
		case line := <-ringbuf:
			content = append(content, line)
		default:
			done = true
		}
	}
	return
}
