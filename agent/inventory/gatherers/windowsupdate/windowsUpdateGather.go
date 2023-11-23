package windowsupdate

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/aliyun/aliyun_assist_client/agent/inventory/model"
	"github.com/aliyun/aliyun_assist_client/agent/log"
	"github.com/aliyun/aliyun_assist_client/common/executil"
)

const (
	// GathererName represents name of windows update gatherer
	GathererName = "ACS:WindowsUpdate"

	SchemaVersionOfWindowsUpdate = "1.0"
	cmd                          = "powershell"
	windowsUpdateQueryCmd        = `
  [Console]::OutputEncoding = [System.Text.Encoding]::UTF8
  Get-WmiObject -Class win32_quickfixengineering | Select-Object HotFixId,Description,@{l="InstalledTime";e={[DateTime]::Parse($_.psbase.properties["installedon"].value,$([System.Globalization.CultureInfo]::GetCultureInfo("en-US"))).ToUniversalTime().ToString("yyyy-MM-ddTHH:mm:ssZ")}},InstalledBy | sort InstalledTime -desc | ConvertTo-Json`
	WindowsUpdateCountLimit         = 500
	WindowsUpdateCountLimitExceeded = "Role Count Limit Exceeded"
)

type T struct{}

// Gatherer returns new windows update gatherer
func Garherer() *T {
	return new(T)
}

// Name returns name of windows update gatherer
func (t *T) Name() string {
	return GathererName
}

// decouple exec.Command for unit test
var cmdExecutor = executeCommand

// Run executes windows update gatherer and returns list of inventory.Item
func (t *T) Run(configuration model.Config) (items []model.Item, err error) {
	log.GetLogger().Info("run windows update gatherer begin")
	var result model.Item
	var data []model.WindowsUpdateData
	out, err := cmdExecutor(cmd, windowsUpdateQueryCmd)
	if err == nil {
		//If there is no windows update in instance, will return empty result instead of throwing error
		if len(out) != 0 {
			err = json.Unmarshal(out, &data)
		}
		if data != nil && len(data) > WindowsUpdateCountLimit {
			err = fmt.Errorf(WindowsUpdateCountLimitExceeded+", got %d", len(data))
			log.GetLogger().WithError(err).Error("gather windows update failed")
			data = []model.WindowsUpdateData{}
		}

		currentTime := time.Now().UTC()
		captureTime := currentTime.Format(time.RFC3339)

		result = model.Item{
			Name:          t.Name(),
			SchemaVersion: SchemaVersionOfWindowsUpdate,
			Content:       data,
			CaptureTime:   captureTime,
		}
		log.GetLogger().Debugf("%v windows update fount", len(data))
		log.GetLogger().Debugf("update info = %+v", result)
	} else {
		log.GetLogger().Errorf("Unable to fetch windows update - %v %v", err.Error(), string(out))
	}
	items = append(items, result)
	log.GetLogger().Infof("run windows update gatherer end, got %d windows updates", len(data))
	return
}

func executeCommand(command string, args ...string) ([]byte, error) {
	return executil.Command(command, args...).CombinedOutput()
}
