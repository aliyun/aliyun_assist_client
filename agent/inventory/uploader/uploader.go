package uploader

import (
	"crypto/md5"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/aliyun/aliyun_assist_client/agent/log"
	"reflect"
	"strings"

	"github.com/aliyun/aliyun_assist_client/agent/inventory/model"
	"github.com/aliyun/aliyun_assist_client/agent/util/jsonutil"
)

const (
	// Name represents name of this component that uploads data to OOS
	Name = "InventoryUploader"
)

type T interface {
	SendDataToOOS(items []*model.InventoryItem) (err error)
	ConvertToOOSInventoryItems(items []model.InventoryItem) (optimizedInventoryItems, nonOptimizedInventoryItems []*model.InventoryItem, err error)
	GetDirtyOOSInventoryItems(items []model.InventoryItem) (dirtyInventoryItems []*model.InventoryItem, err error)
}

// InventoryUploader implements functionality to upload data to OOS Inventory.
type InventoryUploader struct {
	ooscaller OOSCaller
	optimizer Optimizer //helps inventory plugin to optimize PutInventory calls
}

// NewInventoryUploader creates a new InventoryUploader (which sends data to OOS Inventory)
func NewInventoryUploader(instanceID string) (*InventoryUploader, error) {
	var uploader = InventoryUploader{}
	var err error
	if uploader.optimizer, err = NewOptimizerImpl(); err != nil {
		log.GetLogger().Errorf("Unable to load optimizer for inventory uploader because - %v", err.Error())
		return &uploader, err
	}

	if uploader.ooscaller, err = NewOOSCallerImpl(); err != nil {
		log.GetLogger().Errorf("Unable to load ooscaller for inventory uploader because - %v", err.Error())
		return &uploader, err
	}

	return &uploader, nil
}

// SendDataToOOS uploads given inventory items to OOS
func (u *InventoryUploader) SendDataToOOS(instanceID string, items []*model.InventoryItem) (err error) {
	log.GetLogger().Debugf("Uploading following inventory data to OOS - %s %v", instanceID, items)
	log.GetLogger().Infof("Number of Inventory Items: %v", len(items))

	//setting up input for PutInventory API call
	params := &model.PutInventoryInput{
		InstanceId: &instanceID,
		Items:      items,
	}

	err = u.ooscaller.PutInventory(params)
	if err != nil {
		log.GetLogger().Errorf("the following error occured while calling PutInventory API: %v", err)
	} else {
		log.GetLogger().Debug("PutInventory was called successfully")
		u.updateContentHash(items)
	}
	return
}

func (u *InventoryUploader) updateContentHash(items []*model.InventoryItem) {
	log.GetLogger().Debugf("Updating cache")
	for _, item := range items {
		if err := u.optimizer.UpdateContentHash(*item.TypeName, *item.ContentHash); err != nil {
			err = fmt.Errorf("failed to update content hash cache because of - %v", err.Error())
			log.GetLogger().Error(err.Error())
		}
	}
}

// ConvertToOOSInventoryItems converts given array of inventory.Item into an array of *model.InventoryItem. It returns 2 such arrays - one is optimized array
// which contains only contentHash for those inventory types where the dataset hasn't changed from previous collection. The other array is non-optimized array
// which contains both contentHash & content. This is done to avoid iterating over the inventory data twice. It throws error when it encounters error during
// conversion process.
func (u *InventoryUploader) ConvertToOOSInventoryItems(items []model.Item) (optimizedInventoryItems, nonOptimizedInventoryItems []*model.InventoryItem, err error) {

	//NOTE: There can be multiple inventory type data.
	//Each inventory type data => 1 inventory Item. Each inventory type, can contain multiple items

	log.GetLogger().Debugf("Transforming collected inventory data to expected format")

	//iterating over multiple inventory data types.
	for _, item := range items {

		var data string
		var optimizedItem, nonOptimizedItem *model.InventoryItem

		newHash := ""
		oldHash := ""
		itemName := item.Name

		//we should only calculate checksum using content & not include capture time - because that field will always change causing
		//the checksum to change again & again even if content remains same.

		if item.Content == nil || reflect.ValueOf(item.Content).IsNil() {
			data = "[]"
		} else {
			if data, err = jsonutil.Marshal(item.Content); err != nil {
				return
			}
		}

		if data[len(data)-1] == '\n' {
			data = data[0 : len(data)-1]
		}

		newHash = calculateCheckSum([]byte(data))
		log.GetLogger().Debugf("Item being converted - %v with data - %v with checksum - %v", itemName, string(data), newHash)

		//construct non-optimized inventory item
		if nonOptimizedItem, err = convertToOOSInventoryItem(item); err != nil {
			err = fmt.Errorf("formatting inventory data of %v failed due to %v", itemName, err.Error())
			return
		}

		//add contentHash too
		nonOptimizedItem.ContentHash = &newHash

		log.GetLogger().Debugf("NonOptimized item - %+v", nonOptimizedItem)

		nonOptimizedInventoryItems = append(nonOptimizedInventoryItems, nonOptimizedItem)

		//populate optimized item - if content hash matches with earlier collected data.
		oldHash = u.optimizer.GetContentHash(itemName)

		log.GetLogger().Debugf("old hash - %v, new hash - %v for the inventory type - %v", oldHash, newHash, itemName)

		if newHash == oldHash {
			log.GetLogger().Debugf("Inventory data for %v is same as before - we can just send content hash", itemName)

			//set the inventory item accordingly
			optimizedItem = &model.InventoryItem{
				CaptureTime:   &item.CaptureTime,
				TypeName:      &itemName,
				SchemaVersion: &item.SchemaVersion,
				ContentHash:   &oldHash,
			}

			log.GetLogger().Debugf("Optimized item - %v", optimizedItem)

			optimizedInventoryItems = append(optimizedInventoryItems, optimizedItem)

		} else {
			log.GetLogger().Debugf("New inventory data for %v has been detected - can't optimize here", itemName)
			log.GetLogger().Debugf("Adding item - %v to the optimizedItems (since its new data)", nonOptimizedItem)

			optimizedInventoryItems = append(optimizedInventoryItems, nonOptimizedItem)
		}
	}

	return
}

// GetDirtyOOSInventoryItems get the inventory item data for items that have changes since last successful report to OOS.
func (u InventoryUploader) GetDirtyOOSInventoryItems(items []model.Item) (dirtyInventoryItems []*model.InventoryItem, err error) {

	//NOTE: There can be multiple inventory type data.
	//Each inventory type data => 1 inventory Item. Each inventory type, can contain multiple items

	//iterating over multiple inventory data types.
	for _, item := range items {
		var data string
		var rawItem *model.InventoryItem

		newHash := ""
		oldHash := ""
		itemName := item.Name

		//we should only calculate checksum using content & not include capture time - because that field will always change causing
		//the checksum to change again & again even if content remains same.

		if item.Content == nil || reflect.ValueOf(item.Content).IsNil() {
			data = "[]"
		} else {
			if data, err = jsonutil.Marshal(item.Content); err != nil {
				return
			}
		}

		if data[len(data)-1] == '\n' {
			data = data[0 : len(data)-1]
		}

		newHash = calculateCheckSum([]byte(data))
		log.GetLogger().Debugf("Item being converted - %v with data - %v with checksum - %v", itemName, string(data), newHash)

		//construct non-optimized inventory item
		if rawItem, err = convertToOOSInventoryItem(item); err != nil {
			err = fmt.Errorf("Formatting inventory data of %v failed due to %v, rawItem : %#v", itemName, err.Error(), rawItem)
			return
		}

		//add contentHash too
		rawItem.ContentHash = &newHash

		//populate optimized item - if content hash matches with earlier collected data.
		oldHash = u.optimizer.GetContentHash(itemName)

		log.GetLogger().Debugf("Get Dirty inventory items, old hash - %v, new hash - %v for the inventory type - %v", oldHash, newHash, itemName)

		if strings.Compare(newHash, oldHash) != 0 {
			log.GetLogger().Debugf("Dirty inventory type found. Change has been detected for inventory type: %v", itemName)
			dirtyInventoryItems = append(dirtyInventoryItems, rawItem)
		} else {
			log.GetLogger().Debugf("Content hash is the same with the old for %v", itemName)
		}
	}

	return
}

// convertToOOSInventoryItem converts given InventoryItem to []map[string]*string
func convertToOOSInventoryItem(item model.Item) (inventoryItem *model.InventoryItem, err error) {

	var a []interface{}
	var c map[string]*string
	var content = []map[string]*string{}
	var dataB []byte

	dataType := reflect.ValueOf(item.Content)

	switch dataType.Kind() {

	case reflect.Struct:
		//this should be converted to map[string]*string
		c = convertToMap(item.Content)
		content = append(content, c)

	case reflect.Array, reflect.Slice:
		//this should be converted to []map[string]*string
		dataB, _ = json.Marshal(item.Content)
		json.Unmarshal(dataB, &a)

		// If a is empty array, then content has to be empty array
		// instead of nil, as InventoryItem.Content has
		// to be empty array [] after serializing to Json,
		// based on the contract with OOS:PutInventory API.
		for _, v := range a {
			// convert each item to map[string]*string
			c = convertToMap(v)
			content = append(content, c)
		}

	default:
		//NOTE: collected inventory data is expected to be either a struct or an array
		err = fmt.Errorf("Unsupported data format - %v.", dataType.Kind())
		return
	}

	inventoryItem = &model.InventoryItem{
		CaptureTime:   &item.CaptureTime,
		TypeName:      &item.Name,
		SchemaVersion: &item.SchemaVersion,
		Content:       content,
	}

	return inventoryItem, nil
}

// ConvertToMap converts given object to map[string]*string
func convertToMap(input interface{}) (res map[string]*string) {
	var m map[string]interface{}
	b, _ := json.Marshal(input)
	json.Unmarshal(b, &m)

	res = make(map[string]*string)
	for k, v := range m {
		asString := toString(v)
		res[k] = &asString
	}
	return res
}

// toString converts given input to string
func toString(v interface{}) string {
	if v, isString := v.(string); isString {
		return v
	}
	b, _ := json.Marshal(v)
	return string(b)
}

func calculateCheckSum(data []byte) (checkSum string) {
	sum := md5.Sum(data)
	checkSum = base64.StdEncoding.EncodeToString(sum[:])
	return
}
