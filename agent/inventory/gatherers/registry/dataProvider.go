package registry

import (
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/aliyun/aliyun_assist_client/agent/inventory/model"
	"github.com/aliyun/aliyun_assist_client/agent/log"
	"github.com/aliyun/aliyun_assist_client/agent/util/stringutil"
	"github.com/aliyun/aliyun_assist_client/common/executil"

	"github.com/google/uuid"
)

const (
	PowershellCmd      = "powershell"
	MaxValueCountLimit = 250
	ValueLimitExceeded = "ValueLimitExceeded"
)

type filterObj struct {
	Path       string
	Recursive  bool
	ValueNames []string
}

var ValueCountLimitExceeded = errors.New("Exceeded register value count limit")

// LogError is a wrapper on log.Error for easy testability
func LogError(err error) {
	// To debug unit test, please uncomment following line
	// fmt.Println(err)
	log.GetLogger().Error(err)
}

func randomString(length int) string {
	return uuid.New().String()[:length]
}

var cmdExecutor = executeCommand

func executeCommand(command string, args ...string) ([]byte, error) {
	return executil.Command(command, args...).CombinedOutput()
}

// executePowershellCommands executes commands in Powershell to get registry values.
func executePowershellCommands(command, args string) (output []byte, err error) {
	if output, err = cmdExecutor(PowershellCmd, command+" "+args); err != nil {
		log.GetLogger().Errorf("Failed to execute command : %v %v with error - %v",
			command,
			args,
			err.Error())
		log.GetLogger().Debugf("Command Stderr: %v", string(output))
		err = fmt.Errorf("Command failed with error: %v", string(output))
	}

	return
}

func collectDataFromPowershell(powershellCommand string, registryInfo *[]model.RegistryData) (err error) {
	log.GetLogger().Debugf("Executing command: %v", powershellCommand)
	var output []byte
	var cleanOutput string
	output, err = executePowershellCommands(powershellCommand, "")
	if err != nil {
		log.GetLogger().Errorf("Error executing command - %v", err.Error())
		return
	}
	log.GetLogger().Debugf("Before cleanup %v", string(output))
	cleanOutput, err = stringutil.ReplaceMarkedFields(stringutil.CleanupNewLines(string(output)), startMarker, endMarker, stringutil.CleanupJSONField)
	if err != nil {
		LogError(err)
		return
	}

	log.GetLogger().Debugf("Command output: %v", string(cleanOutput))
	if cleanOutput == ValueLimitExceeded {
		log.GetLogger().Error("Number of values collected exceeded limit")
		err = ValueCountLimitExceeded
		return
	}
	if err = json.Unmarshal([]byte(cleanOutput), registryInfo); err != nil {
		err = fmt.Errorf("Unable to parse command output - %v", err.Error())
		log.GetLogger().Error(err.Error())
		log.GetLogger().Debugf("Error parsing command output - no data to return")
	}
	return
}

func collectRegistryData(config model.Config) (data []model.RegistryData, err error) {
	log.GetLogger().Debugf("collectRegistryData called")
	config.Filters = strings.Replace(config.Filters, `\`, `/`, -1)
	var filterList []filterObj
	if err = json.Unmarshal([]byte(config.Filters), &filterList); err != nil {
		return
	}

	valueScanLimit := MaxValueCountLimit

	for _, filter := range filterList {
		var temp []model.RegistryData
		path := filepath.FromSlash(filter.Path)
		recursive := filter.Recursive
		valueNames := filter.ValueNames
		log.GetLogger().Debugf("valueNames %v", valueNames)
		registryPath := "Registry::" + path
		execScript := registryInfoScript + "-Path \"" + registryPath + "\" -ValueLimit " + fmt.Sprint(valueScanLimit)
		if recursive == true {
			execScript += " -Recursive"
		}
		if valueNames != nil && len(valueNames) > 0 {
			valueNamesArg := strings.Join(valueNames, ",")
			execScript += " -Values " + valueNamesArg
		}

		if getRegistryErr := collectDataFromPowershell(execScript, &temp); getRegistryErr != nil {
			LogError(getRegistryErr)
			if getRegistryErr == ValueCountLimitExceeded {
				err = getRegistryErr
				return
			}
			continue
		}

		data = append(data, temp...)
		valueScanLimit = MaxValueCountLimit - len(data)
	}
	log.GetLogger().Debugf("Collected %d registry entries", len(data))
	return
}
