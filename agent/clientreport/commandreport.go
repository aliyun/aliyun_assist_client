package clientreport

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/aliyun/aliyun_assist_client/agent/util/process"
)

type executionResult struct {
	Command      string `json:"command"`
	Result       string `json:"result"`
	ExitCode     int    `json:"exitCode"`
	Output       string `json:"output"`
	ErrorMessage string `json:"errorMessage,omitempty"`
}

// ReportCommandOutput executes specified command and reports its output
func ReportCommandOutput(reportType string, command string, arguments []string) (string, error) {
	var mixedOutput bytes.Buffer
	exitCode, _, err := process.NewProcessCmd().SyncRun("", command, arguments, &mixedOutput, &mixedOutput, nil, nil, 120)

	exitMessage := "Success"
	errorMessage := ""
	if err != nil {
		exitMessage = "Failed"
		errorMessage = err.Error()
	} else if exitCode == process.Fail {
		exitMessage = "Failed"
	} else if exitCode == process.Timeout {
		exitMessage = "Timeout"
	}

	result := executionResult{
		Command:      fmt.Sprintf("%s %s", command, strings.Join(arguments, " ")),
		Result:       exitMessage,
		ExitCode:     exitCode,
		Output:       mixedOutput.String(),
		ErrorMessage: errorMessage,
	}
	resultJSONBytes, err := json.Marshal(result)
	if err != nil {
		return "", err
	}

	report := ClientReport{
		ReportType: reportType,
		Info:       string(resultJSONBytes),
	}
	return SendReport(report)
}
