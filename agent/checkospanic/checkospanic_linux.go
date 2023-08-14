package checkospanic

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/aliyun/aliyun_assist_client/agent/log"
	"github.com/aliyun/aliyun_assist_client/agent/metrics"
	"github.com/aliyun/aliyun_assist_client/agent/util"
	"github.com/aliyun/aliyun_assist_client/thirdparty/sirupsen/logrus"
)

const (
	kdumpPath       = "/var/crash"
	kdumpConfigPath = "/etc/kdump.conf"
	vmcoreDmesgFile = "vmcore-dmesg.txt"
)

var (
	// 127.0.0.1-2023-07-05-20:51:21 or <hostname>-2023-07-05-20:51:21
	vmcorePathRegex      = regexp.MustCompile(`^(?:[\w.-]+)-(\d{4}-\d{2}-\d{2}-\d{2}:\d{2}:\d{2})$`)
	kernelPanicInfoRegex = regexp.MustCompile(`^(Unable to handle kernel|BUG: unable to handle kernel|Kernel BUG at|kernel BUG at|Bad mode in|Oops|Kernel panic)`)
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
	if !util.CheckFileIsExist(vmcoreDmesgPath) {
		logger.Error("vmcore dmesg file not exist", vmcoreDmesgPath)
		return
	}
	if content, err := os.ReadFile(vmcoreDmesgPath); err != nil {
		logger.WithFields(logrus.Fields{
			"file": vmcoreDmesgPath,
			"err":  err,
		}).Error("read vmcore dmesg file failed")
		return
	} else {
		var kernelPanicInfo, rip, callTrace string
		rip, callTrace, kernelPanicInfo = ParseVmcore(string(content))
		metrics.GetLinuxGuestOSPanicEvent(
			"rip", rip,
			"callTrace", callTrace,
			"kernelPanicInfo", kernelPanicInfo,
			"crashTime", latestTime.Format("2006-01-02 15:04:05"),
			"vmcoreDir", vmcoreDir,
		).ReportEvent()
		logger.Info("the latest vmcore file has reported")
	}
}

// ParseVmcore parse fileds `Call Trace` `RIP` `Kernel Panic` from vmcore-dmesg.txt
func ParseVmcore(content string) (rip, callTrace, panicInfo string) {
	var callTraceLines []string
	inCallTrace := false
	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if len(line) == 0 {
			continue
		}
		idx := strings.Index(line, "] ")
		if idx != -1 && idx < len(line)-1 {
			line = line[idx+2:]
		}
		if inCallTrace {
			if line[0] == ' ' {
				callTraceLines = append(callTraceLines, line)
				continue
			} else {
				inCallTrace = false
			}
		}
		if len(callTraceLines) == 0 && strings.HasPrefix(strings.ToLower(line), "call trace:") {
			callTraceLines = append(callTraceLines, "Call Trace:")
			inCallTrace = true
		} else if panicInfo == "" && kernelPanicInfoRegex.MatchString(line) {
			panicInfo = line
		} else if rip == "" && strings.HasPrefix(line, "RIP:") {
			rip = line
		}
	}
	callTrace = strings.Join(callTraceLines, "\n")
	return
}

// FindLocalVmcoreDmesg find latest directory which stores the vmcore-dmesg.txt
func FindLocalVmcoreDmesg(logger logrus.FieldLogger) (vmcoreDmesgPath, latestDir string, latestTime time.Time) {
	kdumpDirTemp := kdumpPath
	// read /etc/kdump.conf to get vmcore directory, default is /var/crash
	if util.CheckFileIsExist(kdumpConfigPath) {
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
	if !util.CheckFileIsExist(kdumpDirTemp) {
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
				vmcoreTime, err := time.Parse("2006-01-02-15:04:05", items[1])
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
