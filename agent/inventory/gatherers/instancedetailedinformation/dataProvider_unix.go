//go:build freebsd || linux || netbsd || openbsd
// +build freebsd linux netbsd openbsd

package instancedetailedinformation

import (
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"

	"github.com/aliyun/aliyun_assist_client/agent/inventory/model"
	"github.com/aliyun/aliyun_assist_client/agent/log"
	"github.com/aliyun/aliyun_assist_client/common/executil"
)

const (
	lscpuCmd          = "lscpu"
	socketsKey        = "Socket(s)"
	coresPerSocketKey = "Core(s) per socket"
	threadsPerCoreKey = "Thread(s) per core"
	cpuModelNameKey   = "Model name"
	cpusKey           = "CPU(s)"
	cpuSpeedMHzKey    = "CPU MHz"
)

// cmdExecutor decouples exec.Command for easy testability
var cmdExecutor = executeCommand

func executeCommand(command string, args ...string) ([]byte, error) {
	cmd := executil.Command(command, args...)
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, "LANG=en_US.UTF-8")
	return cmd.CombinedOutput()
}

// collectPlatformDependentInstanceData collects data from the system.
func collectPlatformDependentInstanceData() (appData []model.InstanceDetailedInformation) {
	var output []byte
	var err error
	cmd := lscpuCmd

	log.GetLogger().Debugf("Executing command: %v", cmd)
	if output, err = cmdExecutor(cmd); err != nil {
		log.GetLogger().Errorf("Failed to execute command : %v; error: %v", cmd, err.Error())
		log.GetLogger().Debugf("Command Stderr: %v", string(output))
		return
	}

	log.GetLogger().Debugf("Parsing output %v", string(output))
	r := parseLscpuOutput(string(output))
	log.GetLogger().Debugf("Parsed output %v", r)
	return r
}

// parseLscpuOutput collects relevant fields from lscpu output, which has the following format (some lines omitted):
//
//	CPU(s):                2
//	Thread(s) per core:    1
//	Core(s) per socket:    2
//	Socket(s):             1
//	Model name:            Intel(R) Xeon(R) CPU E5-2676 v3 @ 2.40GHz
//	CPU MHz:               2400.072
func parseLscpuOutput(output string) (data []model.InstanceDetailedInformation) {
	cpuSpeedMHzStr := getFieldValue(output, cpuSpeedMHzKey)
	if cpuSpeedMHzStr != "" {
		cpuSpeedMHzStr = strconv.Itoa(int(math.Trunc(parseFloat(cpuSpeedMHzStr, 0))))
	}

	socketsStr := getFieldValue(output, socketsKey)

	cpuCoresStr := ""
	coresPerSocketStr := getFieldValue(output, coresPerSocketKey)
	if socketsStr != "" && coresPerSocketStr != "" {
		sockets := parseInt(socketsStr, 0)
		coresPerSocket := parseInt(coresPerSocketStr, 0)
		cpuCoresStr = strconv.Itoa(sockets * coresPerSocket)
	}

	hyperThreadEnabledStr := ""
	threadsPerCoreStr := getFieldValue(output, threadsPerCoreKey)
	if threadsPerCoreStr != "" {
		hyperThreadEnabledStr = boolToStr(parseInt(threadsPerCoreStr, 0) > 1)
	}

	itemContent := model.InstanceDetailedInformation{
		CPUModel:              getFieldValue(output, cpuModelNameKey),
		CPUs:                  getFieldValue(output, cpusKey),
		CPUSpeedMHz:           cpuSpeedMHzStr,
		CPUSockets:            socketsStr,
		CPUCores:              cpuCoresStr,
		CPUHyperThreadEnabled: hyperThreadEnabledStr,
	}

	data = append(data, itemContent)
	return
}

// getFieldValue looks for the first substring of the form "key: value \n" and returns the "value"
// if no such field found, returns empty string
func getFieldValue(input string, key string) string {
	keyStartPos := strings.Index(input, key+":")
	if keyStartPos < 0 {
		return ""
	}

	// add "\n" sentinel in case the key:value pair is on the last line and there is no newline at the end
	afterKey := input[keyStartPos+len(key)+1:] + "\n"
	valueEndPos := strings.Index(afterKey, "\n")
	return strings.TrimSpace(afterKey[:valueEndPos])
}

func parseInt(value string, defaultValue int) int {
	res, err := strconv.Atoi(value)
	if err != nil {
		return defaultValue
	}
	return res
}

func parseFloat(value string, defaultValue float64) float64 {
	res, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return defaultValue
	}
	return res
}

func boolToStr(b bool) string {
	return fmt.Sprintf("%v", b)
}
