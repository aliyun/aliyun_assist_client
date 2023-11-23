//go:build windows
// +build windows

package file

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/aliyun/aliyun_assist_client/agent/util"
	"github.com/aliyun/aliyun_assist_client/agent/util/osutil"
	"github.com/aliyun/aliyun_assist_client/agent/util/stringutil"
	"github.com/aliyun/aliyun_assist_client/common/executil"

	"github.com/aliyun/aliyun_assist_client/agent/log"

	"github.com/aliyun/aliyun_assist_client/agent/inventory/appconfig"
	"github.com/aliyun/aliyun_assist_client/agent/inventory/model"

	"github.com/google/uuid"
)

var (
	startMarker       = "<start" + randomString(8) + ">"
	endMarker         = "<end" + randomString(8) + ">"
	FileInfoBatchSize = 100
	fileInfoScript    = `
  [Console]::OutputEncoding = [System.Text.Encoding]::UTF8
  function getjson($Paths){
	try {
		$a = Get-ItemProperty -Path $Paths -EA SilentlyContinue |
		SELECT-OBJECT Name,Length,VersionInfo,@{n="LastWriteTime";e={[datetime]::ParseExact($_."LastWriteTime","MM/dd/yyyy HH:mm:ss",$null).ToUniversalTime().ToString("yyyy-MM-ddTHH:mm:ssZ")}},
		@{n="CreationTime";e={[datetime]::ParseExact($_."CreationTime","MM/dd/yyyy HH:mm:ss",$null).ToUniversalTime().ToString("yyyy-MM-ddTHH:mm:ssZ")}},
		@{n="LastAccessTime";e={[datetime]::ParseExact($_."LastAccessTime","MM/dd/yyyy HH:mm:ss",$null).ToUniversalTime().ToString("yyyy-MM-ddTHH:mm:ssZ")}},DirectoryName
		$jsonObj = @()
		foreach ($p in $a) {
			$Name = $p.Name
			$Length = $p.Length
			$Description = $p.VersionInfo.FileDescription
			$Version = $p.VersionInfo.FileVersion
			$InstalledDate = $p.CreationTime
			$LastAccesstime = $p.LastAccessTime
			$ProductName = $p.VersionInfo.ProductName
			$ProductVersion = $p.VersionInfo.ProductVersion
			$ProductLanguage = $p.VersionInfo.Language
			$CompanyName = $p.VersionInfo.CompanyName
			$InstalledDir = $p.DirectoryName
			$ModTime = $p.LastWriteTime
			$jsonObj += @"
{"CompanyName": "` + mark(`$CompanyName`) + `", "ProductName": "` + mark(`$ProductName`) + `", "ProductVersion": "$ProductVersion", "ProductLanguage": "$ProductLanguage", "Name":"$Name", "Size":"$Length",
"Description":"` + mark(`$Description`) + `" ,"FileVersion":"$Version","InstalledDate":"$InstalledDate","LastAccessTime":"$LastAccessTime","InstalledDir":"` + mark(`$InstalledDir`) + `","ModificationTime":"$ModTime"}
"@
		}
		$result = $jsonObj -join ","
		$result = "[" + $result + "]"
		[Console]::WriteLine($result)
	} catch {
		Write-Error $_.Exception.Message
	}

}

getjson -Paths `
)

const (
	PowershellCmd  = "powershell"
	SleepTimeMs    = 5000
	ScriptFileName = "getFileInfo.ps1"
)

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func randomString(length int) string {
	return uuid.New().String()[:length]
}

func mark(s string) string {
	return startMarker + s + endMarker
}

var cmdExecutor = executeCommand
var writeFileText = writeFile

func executeCommand(command string, args ...string) ([]byte, error) {
	return executil.Command(command, args...).CombinedOutput()
}

// expand function expands windows environment variables
func expand(s string, mapping func(string) string) (newStr string, err error) {
	newStr, err = stringutil.ReplaceMarkedFields(s, "%", "%", mapping)
	if err != nil {
		return "", err
	}
	return
}

// executePowershellCommands executes commands in Powershell to get all windows files installed.
func executePowershellCommands(command, args string) (output []byte, err error) {
	if output, err = cmdExecutor(PowershellCmd, command+" "+args); err != nil {
		log.GetLogger().Errorf("Failed to execute command : %v %v with error - %v",
			command,
			args,
			err.Error())
		log.GetLogger().Debugf("Failed to execute command : %v %v with error - %v",
			command,
			args,
			err.Error())
		log.GetLogger().Debugf("Command Stderr: %v", string(output))
		err = fmt.Errorf("Command failed with error: %v", string(output))
	}

	return
}

func collectDataFromPowershell(powershellCommand string, fileInfo *[]model.FileData) (err error) {
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
		log.GetLogger().Error(err)
		return
	}
	log.GetLogger().Debugf("Command output: %v", string(cleanOutput))

	if err = json.Unmarshal([]byte(cleanOutput), fileInfo); err != nil {
		err = fmt.Errorf("Unable to parse command output - %v", err.Error())
		log.GetLogger().Error(err.Error())
		log.GetLogger().Debugf("Error parsing command output - no data to return")
	}
	return
}

func writeFile(path string, commands string) (err error) {
	err = osutil.WriteFile(path, commands)
	return
}

// Powershell has limit on number of parameters. So execute command using script.
func createScript(commands string) (path string, err error) {

	if err != nil {
		log.GetLogger().Errorf("Error getting machineID")
		return
	}

	path = filepath.Join(appconfig.DefaultDataStorePath,
		util.GetInstanceId(),
		appconfig.InventoryRootDirName,
		appconfig.FileInventoryRootDirName,
		ScriptFileName)
	log.GetLogger().Debugf("Writing to script file %v", path)

	err = writeFileText(path, commands)
	if err != nil {
		log.GetLogger().Errorf(err.Error())
	}
	return
}

func getPowershellCmd(paths []string) (cmd string, err error) {
	var transformed []string
	for _, x := range paths {
		transformed = append(transformed, `"`+x+`"`)
	}
	cmd = fileInfoScript + strings.Join(transformed, ",")
	return
}

// getMetaData creates powershell script for getting file metadata and executes the script
func getMetaDataForFiles(paths []string) (fileInfo []model.FileData, err error) {
	var cmd string
	cmd, err = getPowershellCmd(paths)
	if err != nil {
		return
	}
	err = collectDataFromPowershell(cmd, &fileInfo)
	return

}

// Tries to create a powershell script and executes it
func createAndRunScript(paths []string) (fileInfo []model.FileData, err error) {
	var cmd, path string
	cmd, err = getPowershellCmd(paths)
	if err != nil {
		log.GetLogger().Errorf(err.Error())
		return
	}
	path, err = createScript(cmd)
	if err != nil {
		log.GetLogger().Errorf(err.Error())
		return
	}

	powershellArg := "& '" + path + "'"
	log.GetLogger().Debugf("Executing command %v", powershellArg)
	err = collectDataFromPowershell(powershellArg, &fileInfo)

	osutil.DeleteFile(path)
	return
}

// Its is more efficient to run using script. So try to run command using script.
// If there is an error we should try fallback method.
func getMetaData(paths []string) (fileInfo []model.FileData, err error) {
	var batchPaths []string

	var scriptErr error
	fileInfo, scriptErr = createAndRunScript(paths)

	// If err running the script, try fallback option
	if scriptErr != nil {
		for i := 0; i < len(paths); i += FileInfoBatchSize {
			batchPaths = paths[i:min(i+FileInfoBatchSize, len(paths))]
			fileInfoBatch, metaDataErr := getMetaDataForFiles(batchPaths)
			if metaDataErr != nil {
				log.GetLogger().Error(metaDataErr)
				err = metaDataErr
				return
			}
			fileInfo = append(fileInfo, fileInfoBatch...)
		}
		return
	}
	err = scriptErr
	return
}
