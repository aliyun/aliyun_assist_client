package role

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/aliyun/aliyun_assist_client/agent/util"
	"github.com/aliyun/aliyun_assist_client/agent/util/osutil"
	"github.com/aliyun/aliyun_assist_client/agent/util/stringutil"

	"github.com/aliyun/aliyun_assist_client/agent/inventory/appconfig"
	"github.com/aliyun/aliyun_assist_client/agent/inventory/model"

	"github.com/aliyun/aliyun_assist_client/agent/log"

	"github.com/google/uuid"
)

var (
	startMarker    = "<start" + randomString(8) + ">"
	endMarker      = "<end" + randomString(8) + ">"
	roleInfoScript = `
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8
import-module ServerManager
$roleInfo = Get-WindowsFeature | Select-Object Name, DisplayName, Description, Installed, InstallState, FeatureType, Path, SubFeatures, ServerComponentDescriptor, DependsOn, Parent
$jsonObj = @()
foreach($r in $roleInfo) {
$Name = $r.Name
$DisplayName = $r.DisplayName
$Description = $r.Description
$Installed = $r.Installed
$InstalledState = $r.InstallState
$FeatureType = $r.FeatureType
$Path = $r.Path
$SubFeatures = $r.SubFeatures
$ServerComponentDescriptor = $r.ServerComponentDescriptor
$DependsOn = $r.DependsOn
$Parent = $r.Parent
$jsonObj += @"
{"Name": "` + mark(`$Name`) + `", "DisplayName": "` + mark(`$DisplayName`) + `", "Description": "` + mark(`$Description`) + `", "Installed": "$Installed",
"InstalledState": "$InstalledState", "FeatureType": "$FeatureType", "Path": "` + mark(`$Path`) + `", "SubFeatures": "` + mark(`$SubFeatures`) + `", "ServerComponentDescriptor": "` + mark(`$ServerComponentDescriptor`) + `", "DependsOn": "` + mark(`$DependsOn`) + `", "Parent": "` + mark(`$Parent`) + `"}
"@
}
$result = $jsonObj -join ","
$result = "[" + $result + "]"
[Console]::WriteLine($result)
`
	roleInfoScriptUsingRegistry = `
  [Console]::OutputEncoding = [System.Text.Encoding]::UTF8
	$keyExists = Test-Path "Registry::HKEY_LOCAL_MACHINE\SOFTWARE\Microsoft\Windows\CurrentVersion\Setup\OC Manager\Subcomponents"
	$jsonObj = @()
	if ($keyExists) {
		$key = Get-Item "Registry::HKEY_LOCAL_MACHINE\SOFTWARE\Microsoft\Windows\CurrentVersion\Setup\OC Manager\Subcomponents"
		$valueNames = $key.GetValueNames();
		foreach ($valueName in $valueNames) {
			$value = $key.GetValue($valueName);
			if ($value -gt 0) {
				$installed = "True"
			} else {
				$installed = "False"
			}
			$jsonObj += @"
{"Name": "$valueName", "Installed": "$installed"}
"@

		}
	}
	$result = $jsonObj -join ","
	$result = "[" + $result + "]"
	[Console]::WriteLine($result)
`
)

const (
	// Use powershell to get role info
	PowershellCmd = "powershell"
	QueryFileName = "roleInfo.xml"
)

type RoleService struct {
	RoleService []RoleService
	DisplayName string `xml:"DisplayName,attr"`
	Installed   string `xml:"Installed,attr"`
	Id          string `xml:"Id,attr"`
	Default     string `xml:"Default,attr"`
}

type Role struct {
	RoleService []RoleService
	DisplayName string `xml:"DisplayName,attr"`
	Installed   string `xml:"Installed,attr"`
	Id          string `xml:"Id,attr"`
	Default     string `xml:"Default,attr"`
}

type Feature struct {
	Feature     []Feature
	DisplayName string `xml:"DisplayName,attr"`
	Installed   string `xml:"Installed,attr"`
	Id          string `xml:"Id,attr"`
	Default     string `xml:"Default,attr"`
}

type Result struct {
	Role    []Role
	Feature []Feature
}

func randomString(length int) string {
	return uuid.New().String()[:length]
}

func mark(s string) string {
	return startMarker + s + endMarker
}

// LogError is a wrapper on log.Error for easy testability
func LogError(err error) {
	// To debug unit test, please uncomment following line
	// fmt.Println(err)
	log.GetLogger().Error(err)
}

var cmdExecutor = executeCommand
var readFile = readAllText
var resultPath = getResultFilePath

func executeCommand(command string, args ...string) ([]byte, error) {
	return exec.Command(command, args...).CombinedOutput()
}

func readAllText(path string) (xmlData string, err error) {
	xmlData, err = osutil.ReadFile(path)
	return
}

func getResultFilePath() (path string, err error) {
	path = filepath.Join(appconfig.DefaultDataStorePath,
		util.GetInstanceId(),
		appconfig.InventoryRootDirName,
		appconfig.RoleInventoryRootDirName,
		QueryFileName)
	return
}

func readServiceData(roleService RoleService, roleInfo *[]model.RoleData) {
	roleData := model.RoleData{
		Name:        roleService.Id,
		DisplayName: roleService.DisplayName,
		Installed:   strings.Title(roleService.Installed),
		FeatureType: "Role Service",
	}
	*roleInfo = append(*roleInfo, roleData)
	for i := 0; i < len(roleService.RoleService); i++ {
		readServiceData(roleService.RoleService[i], roleInfo)
	}
}

func readRoleData(role Role, roleInfo *[]model.RoleData) {
	roleData := model.RoleData{
		Name:        role.Id,
		DisplayName: role.DisplayName,
		Installed:   strings.Title(role.Installed),
		FeatureType: "Role",
	}
	*roleInfo = append(*roleInfo, roleData)

	for i := 0; i < len(role.RoleService); i++ {
		readServiceData(role.RoleService[i], roleInfo)
	}
}

func readFeatureData(feature Feature, roleInfo *[]model.RoleData) {

	roleData := model.RoleData{
		Name:        feature.Id,
		DisplayName: feature.DisplayName,
		Installed:   strings.Title(feature.Installed),
		FeatureType: "Feature",
	}
	*roleInfo = append(*roleInfo, roleData)

	for i := 0; i < len(feature.Feature); i++ {
		readFeatureData(feature.Feature[i], roleInfo)
	}
}

func readAllData(result Result, roleInfo *[]model.RoleData) {
	roles := result.Role
	features := result.Feature

	for i := 0; i < len(roles); i++ {
		readRoleData(roles[i], roleInfo)
	}

	for i := 0; i < len(features); i++ {
		readFeatureData(features[i], roleInfo)
	}
}

// executePowershellCommands executes commands in Powershell to get all windows processes.
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

func collectDataFromPowershell(powershellCommand string, roleInfo *[]model.RoleData) (err error) {
	log.GetLogger().Debugf("Executing command: %v", powershellCommand)
	var output []byte
	var cleanOutput string
	output, err = executePowershellCommands(powershellCommand, "")
	if err != nil {
		log.GetLogger().Errorf("Error executing command - %v", err.Error())
		return
	}
	log.GetLogger().Debugf("Command output before clean up: %v", string(output))

	cleanOutput, err = stringutil.ReplaceMarkedFields(stringutil.CleanupNewLines(string(output)), startMarker, endMarker, stringutil.CleanupJSONField)
	if err != nil {
		LogError(err)
		return
	}
	log.GetLogger().Debugf("Command output: %v", string(cleanOutput))

	if err = json.Unmarshal([]byte(cleanOutput), roleInfo); err != nil {
		err = fmt.Errorf("Unable to parse command output - %v", err.Error())
		log.GetLogger().Error(err.Error())
		log.GetLogger().Debugf("Error parsing command output - no data to return")
	}
	return
}

// Some early 2008 versions use ServerManager for role management, so use that for collecting data.
func collectDataUsingServerManager(roleInfo *[]model.RoleData) (err error) {
	var xmlData, path string
	var output []byte

	path, err = resultPath()

	if err != nil {
		log.GetLogger().Errorf("Error getting path of file")
		return
	}

	powershellCommand := "Servermanagercmd.exe -q " + path
	output, err = executePowershellCommands(powershellCommand, "")
	log.GetLogger().Debugf("Command output: %v", string(output))

	if err != nil {
		log.GetLogger().Errorf("Error executing command - %v", err.Error())
		return
	}

	xmlData, err = readFile(path)
	if err != nil {
		log.GetLogger().Errorf("Error reading role info file - %v", err.Error())
		return
	}

	v := Result{}
	err = xml.Unmarshal([]byte(xmlData), &v)
	if err != nil {
		log.GetLogger().Errorf("Error unmarshalling xml: %v", err.Error())
		return
	}

	readAllData(v, roleInfo)
	osutil.DeleteFile(path)
	return
}

func collectRoleData(config model.Config) (data []model.RoleData, err error) {
	log.GetLogger().Debugf("collectRoleData called")

	err = collectDataFromPowershell(roleInfoScript, &data)
	// Some early 2008 releases uses server manager for getting role information
	if err != nil {
		log.GetLogger().Debugf("Trying collecting role data using server manager")
		err = collectDataUsingServerManager(&data)
	}
	// In some versions of 2003, roles information is stored as subcomponents in registry.
	if err != nil {
		log.GetLogger().Debugf("Trying collecting role data using registry")
		err = collectDataFromPowershell(roleInfoScriptUsingRegistry, &data)
	}
	if err == nil && data != nil && len(data) > RoleCountLimit {
		err = fmt.Errorf(RoleCountLimitExceeded+", got %d", len(data))
		log.GetLogger().WithError(err).Error("collect role data failed")
		return []model.RoleData{}, err
	}
	if err != nil {
		log.GetLogger().Errorf("Failed to collect role data using possible mechanisms")
	}
	return
}
