package service

import (
	"encoding/json"
	"fmt"
	"os/exec"

	"github.com/aliyun/aliyun_assist_client/agent/log"
	"github.com/aliyun/aliyun_assist_client/agent/util/stringutil"

	"github.com/aliyun/aliyun_assist_client/agent/inventory/model"

	"github.com/google/uuid"
)

var (
	startMarker       = "<start" + randomString(8) + ">"
	endMarker         = "<end" + randomString(8) + ">"
	serviceInfoScript = `
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8
$serviceInfo = Get-Service | Select-Object Name, DisplayName, Status, DependentServices, ServicesDependedOn, ServiceType, StartType
$jsonObj = @()
foreach($s in $serviceInfo) {
$Name = $s.Name
$DisplayName = $s.DisplayName
$Status = $s.Status
$DependentServices = $s.DependentServices
$ServicesDependedOn = $s.ServicesDependedOn
$ServiceType = $s.ServiceType
$StartType = $s.StartType
$jsonObj += @"
{"Name": "` + mark(`$Name`) + `", "DisplayName": "` + mark(`$DisplayName`) + `", "Status": "$Status", "DependentServices": "` + mark(`$DependentServices`) + `",
"ServicesDependedOn": "` + mark(`$ServicesDependedOn`) + `", "ServiceType": "$ServiceType", "StartType": "$StartType"}
"@
}
$result = $jsonObj -join ","
$result = "[" + $result + "]"
[Console]::WriteLine($result)
`
)

const (
	PowerShellCmd = "powershell"
)

func randomString(length int) string {
	return uuid.New().String()[:length]
}

func mark(s string) string {
	return startMarker + s + endMarker
}

var cmdExecutor = executeCommand

func executeCommand(command string, args ...string) ([]byte, error) {
	return exec.Command(command, args...).CombinedOutput()
}

func executePowershellCommands(command, args string) (output []byte, err error) {
	if output, err = cmdExecutor(PowerShellCmd, command+" "+args); err != nil {
		log.GetLogger().Debugf("Failed to execute command: %v %v with error - %v",
			command,
			args,
			err.Error())
		log.GetLogger().Debugf("Command Stderr: %v", string(output))
		err = fmt.Errorf("Command failed with error: %v", string(output))
	}
	return
}

func collectDataFromPowerShell(powershellCommand string, serviceInfo *[]model.ServiceData) (err error) {
	var output []byte
	var cleanOutPut string
	log.GetLogger().Debugf("Executing command: %v", powershellCommand)
	output, err = executePowershellCommands(powershellCommand, "")
	if err != nil {
		log.GetLogger().Errorf("Error executing command - %v", err.Error())
		return
	}
	log.GetLogger().Debugf("Command output before clean up: %v", string(cleanOutPut))
	cleanOutPut, err = stringutil.ReplaceMarkedFields(stringutil.CleanupNewLines(string(output)), startMarker, endMarker, stringutil.CleanupJSONField)
	if err != nil {
		log.GetLogger().Error(err)
	}
	log.GetLogger().Debugf("Command output: %v", string(cleanOutPut))

	if err = json.Unmarshal([]byte(cleanOutPut), serviceInfo); err != nil {
		err = fmt.Errorf("Unable to parse command output - %v", err.Error())
		log.GetLogger().Errorf(err.Error())
		log.GetLogger().Debugf("Error parsing command output - no data to return")
	}
	if serviceInfo != nil && len(*serviceInfo) > ServiceCountLimit {
		err = fmt.Errorf(ServiceCountLimitExceeded+", got %d", len(*serviceInfo))
		return
	}
	return
}

func collectServiceData(config model.Config) (data []model.ServiceData, err error) {
	log.GetLogger().Debugf("collectServiceData called")
	err = collectDataFromPowerShell(serviceInfoScript, &data)
	if err != nil {
		log.GetLogger().WithError(err).Error("collect service failed")
		return []model.ServiceData{}, err
	}
	return
}
