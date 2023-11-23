//go:build windows
// +build windows

package instancedetailedinformation

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/aliyun/aliyun_assist_client/agent/inventory/model"
	"github.com/aliyun/aliyun_assist_client/agent/log"
	"github.com/aliyun/aliyun_assist_client/common/executil"
)

const (
	PowershellCmd = "powershell"
	CPUInfoScript = `
$wmi_proc = Get-WmiObject -Class Win32_Processor
if (@($wmi_proc)[0].NumberOfCores) #Modern OS
{
    $Sockets = @($wmi_proc).Count
    $Cores = ($wmi_proc | Measure-Object -Property NumberOfCores -Sum).Sum
    $CPUs = ($wmi_proc | Measure-Object -Property NumberOfLogicalProcessors -Sum).Sum

}
else #Legacy OS
{
    $Sockets = @($wmi_proc | Select-Object -Property SocketDesignation -Unique).Count
    $Cores = @($wmi_proc).Count
    $CPUs=$Cores
}
$CPUModel=@($wmi_proc)[0].Name
$CPUSpeed=@($wmi_proc)[0].MaxClockSpeed
if ($Cores -lt $CPUs) {
    $Hyperthread="true"
} else {
    $Hyperthread="false"
}
Write-Host -nonewline @"
{"CPUModel":"$CPUModel","CPUSpeedMHz":"$CPUSpeed","CPUs":"$CPUs","CPUSockets":"$Sockets","CPUCores":"$Cores","CPUHyperThreadEnabled":"$HyperThread"}
"@`
	OsInfoScript = `GET-WMIOBJECT -class win32_operatingsystem |
SELECT-OBJECT ServicePackMajorVersion,BuildNumber | % { Write-Output @"
{"OSServicePack":"$($_.ServicePackMajorVersion)"}
"@}`
)

// decoupling exec.Command for easy testability
var cmdExecutor = executeCommand

func executeCommand(command string, args ...string) ([]byte, error) {
	return executil.Command(command, args...).CombinedOutput()
}

// collectPlatformDependentInstanceData collects data from the system.
func collectPlatformDependentInstanceData() (appData []model.InstanceDetailedInformation) {
	log.GetLogger().Debugf("Getting %v data", GathererName)
	var instanceDetailedInfo model.InstanceDetailedInformation
	err1 := collectDataFromPowershell(CPUInfoScript, &instanceDetailedInfo)
	err2 := collectDataFromPowershell(OsInfoScript, &instanceDetailedInfo)
	if err1 != nil && err2 != nil {
		// if both commands fail, return no data
		return
	}
	appData = append(appData, instanceDetailedInfo)
	str, _ := json.Marshal(appData)
	log.GetLogger().Debugf("%v gathered: %v", GathererName, string(str))
	return
}

func collectDataFromPowershell(powershellCommand string, instanceDetailedInfoResult *model.InstanceDetailedInformation) (err error) {
	log.GetLogger().Debugf("Executing command: %v", powershellCommand)
	output, err := executePowershellCommands(powershellCommand, "")
	if err != nil {
		log.GetLogger().Errorf("Error executing command - %v", err.Error())
		return
	}
	output = []byte(cleanupNewLines(string(output)))
	log.GetLogger().Infof("Command output: %v", string(output))

	if err = json.Unmarshal([]byte(output), instanceDetailedInfoResult); err != nil {
		err = fmt.Errorf("Unable to parse command output - %v", err.Error())
		log.GetLogger().Error(err.Error())
		log.GetLogger().Infof("Error parsing command output - no data to return")
	}
	return
}

func cleanupNewLines(s string) string {
	return strings.Replace(strings.Replace(s, "\n", "", -1), "\r", "", -1)
}

// executePowershellCommands executes commands in powershell to get all  applications installed.
func executePowershellCommands(command, args string) (output []byte, err error) {
	if output, err = cmdExecutor(PowershellCmd, command+" "+args); err != nil {
		log.GetLogger().Debugf("Failed to execute command : %v %v with error - %v",
			command,
			args,
			err.Error())
		log.GetLogger().Debugf("Command Stderr: %v", string(output))
		err = fmt.Errorf("Command failed with error: %v", string(output))
	}

	return
}
