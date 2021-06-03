package uploader

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"sync"

	"github.com/aliyun/aliyun_assist_client/agent/inventory/appconfig"
	"github.com/aliyun/aliyun_assist_client/agent/util"
	"github.com/aliyun/aliyun_assist_client/agent/util/osutil"

	"github.com/aliyun/aliyun_assist_client/agent/log"
)

var (
	lock             sync.RWMutex
	contentHashStore map[string]string
)

// Optimizer defines operations of content optimizer which inventory plugin makes use of
type Optimizer interface {
	UpdateContentHash(inventoryItemName, hash string) (err error)
	GetContentHash(inventoryItemName string) (hash string)
}

type Impl struct {
	location string //where the content hash data is persisted in file-systems
}

func NewOptimizerImpl() (*Impl, error) {
	return NewOptimizerImplWithLocation(util.GetInstanceId(), appconfig.InventoryRootDirName, appconfig.InventoryContentHashFileName)
}

func NewOptimizerImplWithLocation(instanceId string, rootDir string, fileName string) (*Impl, error) {
	var optimizer = Impl{}
	var content string
	var err error

	optimizer.location = filepath.Join(appconfig.DefaultDataStorePath,
		instanceId,
		rootDir,
		fileName)

	contentHashStore = make(map[string]string)

	//read old content hash values from file
	if osutil.Exists(optimizer.location) {
		log.GetLogger().Debugf("Found older set of content hash used by inventory plugin - %v", optimizer.location)

		//read file
		if content, err = osutil.ReadFile(optimizer.location); err == nil {
			log.GetLogger().Debugf("Found older set of content hash used by inventory plugin at %v - \n%v",
				optimizer.location,
				content)

			if err = json.Unmarshal([]byte(content), &contentHashStore); err != nil {
				log.GetLogger().Debugf("Unable to read content hash store of inventory plugin - thereby ignoring any older values")
			}
		}
	}

	return &optimizer, nil
}

func (i *Impl) UpdateContentHash(inventoryItemName, hash string) (err error) {
	lock.Lock()
	defer lock.Unlock()

	contentHashStore[inventoryItemName] = hash

	//persist the data in file system
	dataB, _ := json.Marshal(contentHashStore)

	if err = osutil.WriteFile(i.location, string(dataB)); err != nil {
		err = fmt.Errorf("Unable to update content hash in file - %v because - %v", i.location, err.Error())
		return
	}

	return
}

func (i *Impl) GetContentHash(inventoryItemName string) (hash string) {
	lock.RLock()
	defer lock.RUnlock()

	var found bool

	if hash, found = contentHashStore[inventoryItemName]; !found {
		// return empty string - if there is no content hash for given inventory data type
		hash = ""
	}

	return
}
