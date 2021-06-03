package inventory

import (
	"fmt"
	"strings"

	"github.com/aliyun/aliyun_assist_client/agent/inventory/gatherers"
	"github.com/aliyun/aliyun_assist_client/agent/inventory/model"
	"github.com/aliyun/aliyun_assist_client/agent/inventory/uploader"
	"github.com/aliyun/aliyun_assist_client/agent/log"
	"github.com/aliyun/aliyun_assist_client/agent/util"
	"github.com/aliyun/aliyun_assist_client/agent/util/jsonutil"
	"github.com/aliyun/aliyun_assist_client/agent/util/osutil"
)

const (
	errorMsgForMultipleAssociations           = "%v detected multiple inventory configurations associated with one instance. Each instance can be associated with just one inventory configuration. Conflicting inventory configuration IDs: %v and %v"
	errorMsgForInvalidInventoryInput          = "invalid or unrecognized input was received for %v plugin"
	errorMsgForExecutingInventoryViaAssociate = "%v plugin can only be invoked via oos-associate"
	errorMsgForUnableToDetectInvocationType   = "it could not be detected if %v plugin was invoked via oos-associate because - %v"
	errorMsgForInabilityToSendDataToOOS       = "inventory data could not be uploaded to server. Additional troubleshooting information - %v"
	msgWhenNoDataToReturnForInventoryPlugin   = "Inventory policy has been successfully applied but there is no inventory data to upload to OOS"
	successfulMsgForInventoryPlugin           = "Inventory policy has been successfully applied and collected inventory data has been uploaded to OOS"
)

var (
	windowsOnlyTypes = []string{"ACS:Service", "ACS:WindowsRole", "ACS:WindowsRegistry", "ACS:WindowsUpdate"}
)

func RunGatherers(policy model.Policy) (items []model.Item, err error) {
	_, installedGatherers := gatherers.InitializeGatherers()
	applyGathererNames := collectGathererNames(policy)
	var applyGatherers []gatherers.T
	if len(applyGathererNames) > 0 {
		for _, applyGathererName := range applyGathererNames {
			if installedGatherers[applyGathererName] == nil {
				log.GetLogger().Errorf(errorMsgForUnableToDetectInvocationType, applyGathererName, "its not exist.")
			} else {
				applyGatherers = append(applyGatherers, installedGatherers[applyGathererName])
			}
		}
	}

	log.GetLogger().Infof("apply inventory gatherers: %v", applyGathererNames)
	if len(applyGatherers) > 0 {
		for _, applyGatherer := range applyGatherers {
			gItems, err := applyGatherer.Run(policy.InventoryPolicy[applyGatherer.Name()])
			if err == nil {
				items = append(items, gItems...)
			} else {
				log.GetLogger().WithError(err).Errorf("run gatherer %s fail", applyGatherer.Name())
			}
		}
	}

	//check if there is data to send to OOS
	if len(items) == 0 {
		//no data to send to OOS - no need to call PutInventory API
		log.GetLogger().Info(msgWhenNoDataToReturnForInventoryPlugin)
		return
	}

	d, _ := jsonutil.Marshal(items)
	log.GetLogger().Debugf("Collected Inventory data: %v", string(d))

	var inventoryUploader *uploader.InventoryUploader
	if inventoryUploader, err = uploader.NewInventoryUploader(util.GetInstanceId()); err != nil {
		err = fmt.Errorf("Unable to configure OOS Inventory uploader - %v", err.Error())
		return
	}
	var optimizedInventoryItems, nonOptimizedInventoryItems []*model.InventoryItem
	if optimizedInventoryItems, nonOptimizedInventoryItems, err = inventoryUploader.ConvertToOOSInventoryItems(items); err != nil {
		err = fmt.Errorf("Encountered error in converting data to OOS InventoryItems - %v. Skipping upload to OOS", err.Error())
		return
	}
	//first send data in optimized fashion
	if err = inventoryUploader.SendDataToOOS(util.GetInstanceId(), optimizedInventoryItems); err != nil {
		log.GetLogger().WithError(err).Error("failed to upload optimized inventory items")
		err = fmt.Errorf("Encountered error in sending data to OOS InventoryItems - %v", err.Error())
		if shouldRetryWithNonOptimizedData(err) {
			//call putinventory again with non-optimized dataset
			if err = inventoryUploader.SendDataToOOS(util.GetInstanceId(), nonOptimizedInventoryItems); err != nil {
				log.GetLogger().WithError(err).Error("failed to upload nonOptimized inventory items")
				//sending non-optimized data also failed
				return
			}
		}
		return
	}
	return
}

func shouldRetryWithNonOptimizedData(err error) bool {
	msg := err.Error()
	if strings.Contains(msg, "Inventory.ItemContentMismatch") || strings.Contains(msg, "Inventory.InvalidItemContent") {
		log.GetLogger().Infof("%v encountered - will try sending nonOptimized inventory data", err.Error())
		return true
	}
	return false
}

func supportByOs(gathererName string) bool {
	osType := osutil.GetOsType()
	for _, dataType := range windowsOnlyTypes {
		if dataType == gathererName && osType != osutil.OSWin {
			return false
		}
	}
	return true
}

func collectGathererNames(policy model.Policy) []string {
	var applyGatherNames []string
	for gathererName, config := range policy.InventoryPolicy {
		if config.Collection == "Enabled" {
			if supportByOs(gathererName) {
				applyGatherNames = append(applyGatherNames, gathererName)
			} else {
				log.GetLogger().Debugf("%s not supported by current os type %s, ignore", gathererName, osutil.GetOsType())
			}
		}
	}
	return applyGatherNames
}
