package pluginmanager

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"io"
	"os"

	"github.com/aliyun/aliyun_assist_client/thirdparty/sirupsen/logrus"
	bolt "go.etcd.io/bbolt"

	"github.com/aliyun/aliyun_assist_client/agent/log"
	"github.com/aliyun/aliyun_assist_client/common/fuzzyjson"
)

const (
	_PluginListBucketName = "pluginList"
)

var (
	errEmptyInstalledPluginsJsonFile = errors.New("empty content in existed installed_plugins JSON database file")
)

// _InstalledPluginsJson 和installed_plugins文件内容一致，用于解析json
type _InstalledPluginsJson struct {
	PluginList []PluginInfo `json:"pluginList"`
}

type InstalledPlugins struct {
	boltdb *bolt.DB
}

// TODO-FIXME: Should LoadInstalledPlugins has a timeout limit? Now it simply
// waits indefinitely.
func LoadInstalledPlugins() (*InstalledPlugins, error) {
	boltPath, err := getInstalledPluginsBoltPath()
	if err != nil {
		return nil, err
	}
	// 1. Just use the new BoltDB file if existed, but NEVER AUTO-CREATE IT if
	// not existed
	boltdb, err := bolt.Open(boltPath, os.FileMode(0o0640), &bolt.Options{
		OpenFile: func(name string, flag int, perm os.FileMode) (*os.File, error) {
			return os.OpenFile(name, flag & ^os.O_CREATE, perm)
		},
	})
	if err == nil {
		return &InstalledPlugins{
			boltdb: boltdb,
		}, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}

	// 2. Just use the new BoltDB file when old JSON file does not exist
	jsonPath, err := getInstalledPluginsJSONPath()
	if err != nil {
		return nil, err
	}
	jsonFile, err := os.OpenFile(jsonPath, os.O_RDONLY, 0)
	if err != nil {
		// Resolved: Simply use BoltDB database file on error encountered when
		// opening installed_plugins JSON database file. Just log the error and
		// let it go then.
		log.GetLogger().WithError(err).Warningln("Failed to open obsolete installed_plugins JSON database file")
		return _loadInstalledPluginsBolt(boltPath)
	}

	// 3. Now migrate data from old existed JSON file to the new BoltDB file,
	// which MUST be created EXCLUSIVELY BY CURRENT PROCESS
	installedPluginInfoBytes, err := _loadEachInstalledPluginJsonAsBytes(jsonFile)
	if err != nil {
		// Resolved: Simply use BoltDB database file on error encountered when
		// reading and parsing installed_plugins JSON database file. Empty file
		// is also unacceptable.
		log.GetLogger().WithError(err).Warningln("Failed to read obsolete installed_plugins JSON database file")
		return _loadInstalledPluginsBolt(boltPath)
	}

	boltdb, err = bolt.Open(boltPath, os.FileMode(0o0640), &bolt.Options{
		OpenFile: func(name string, flag int, perm os.FileMode) (*os.File, error) {
			return os.OpenFile(name, flag | os.O_EXCL, perm)
		},
	})
	if err != nil {
		if errors.Is(err, os.ErrExist) {
			// !!!RACE CONDITION HERE!!!
			// Existed installed_plugins.db BoltDB file means another process
			// (e.g., agent or yet another acs-plugin-manager) is doing or has
			// done the migration. Simply fallback to general opening procedure.
			return _loadInstalledPluginsBolt(boltPath)
		} else {
			return nil, err
		}
	}

	if err := _insertInstalledPluginInfoBytes(boltdb, installedPluginInfoBytes); err != nil {
		_ = boltdb.Close()
		return nil, err
	}

	return &InstalledPlugins{
		boltdb: boltdb,
	}, nil
}

func _loadInstalledPluginsBolt(boltPath string) (*InstalledPlugins, error) {
	boltdb, err := bolt.Open(boltPath, os.FileMode(0o0640), nil)
	if err != nil {
		return nil, err
	}

	return &InstalledPlugins{
		boltdb: boltdb,
	}, nil
}

func _loadEachInstalledPluginJsonAsBytes(jsonFile *os.File) ([][]byte, error) {
	body, err := io.ReadAll(jsonFile)
	if err != nil {
		return nil, err
	}

	content := string(body)
	if len(content) == 0 {
		return nil, errEmptyInstalledPluginsJsonFile
	}

	installedPluginsJson := _InstalledPluginsJson{}
	if err := fuzzyjson.Unmarshal(content, &installedPluginsJson); err != nil {
		log.GetLogger().WithFields(logrus.Fields{
			"content": content,
		}).WithError(err).Errorln("Failed to unmarshal JSON database file of install plugins")
		return nil, err
	}

	var payloads [][]byte
	for _, plugin := range installedPluginsJson.PluginList {
		payload, err := json.Marshal(plugin)
		if err != nil {
			return nil, err
		}

		payloads = append(payloads, payload)
	}

	return payloads, nil
}

func _insertInstalledPluginInfoBytes(boltdb *bolt.DB, payloads [][]byte) error {
	return boltdb.Update(func(tx *bolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists([]byte(_PluginListBucketName))
		if err != nil {
			return err
		}

		for _, payload := range payloads {
			// Generate auto-incremental ID for the plugin information.
			// According to the documentation of https://github.com/etcd-io/bbolt,
			// the NextSequence() method returns an error only if the Tx is closed
			// or not writeable. That can't happen in an Update() call, so the error
			// check can be safely ignored.
			id, _ := bucket.NextSequence()
			insertedKey := int(id)

			if err := bucket.Put(_itob(insertedKey), payload); err != nil {
				return err
			}
		}

		return nil
	})
}

// Close can be called multiple times and internal implementation in bbolt would
// keep it safe. Thus close the database as soon as possible please.
func (ip *InstalledPlugins) Close() error {
	return ip.boltdb.Close()
}

func (ip *InstalledPlugins) FindAll() ([]int, []PluginInfo, error) {
	var keys []int
	var values []PluginInfo

	err := ip.boltdb.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(_PluginListBucketName))
		if bucket == nil {
			return nil
		}

		return bucket.ForEach(func(k, v []byte) error {
			var plugin PluginInfo
			if err := json.Unmarshal(v, &plugin); err != nil {
				// Raising errors to caller for more flexible handling
				return err
			}

			keys = append(keys, _b8toi(k))
			values = append(values, plugin)
			return nil
		})
	})
	if err != nil {
		return nil, nil, err
	}

	return keys, values, nil
}

func (ip *InstalledPlugins) FindManyByName(name string) ([]int, []PluginInfo, error) {
	keys, values, err := ip.FindAll()
	if err != nil {
		return nil, nil, err
	}

	foundKeys := []int{}
	foundValues := []PluginInfo{}
	for i := 0; i < len(values); i++ {
		if values[i].Name != name {
			continue
		}

		foundKeys = append(foundKeys, keys[i])
		foundValues = append(foundValues, values[i])
	}

	return foundKeys, foundValues, nil
}

func (ip *InstalledPlugins) FindOneWithPredicate(predicate func (plugin *PluginInfo) bool) (int, *PluginInfo, error) {
	var foundKey int = -1
	var foundValue *PluginInfo

	err := ip.boltdb.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(_PluginListBucketName))
		if bucket == nil {
			return nil
		}

		cursor := bucket.Cursor()
		for k, v := cursor.First(); k != nil; k, v = cursor.Next() {
			var plugin PluginInfo
			if err := json.Unmarshal(v, &plugin); err != nil {
				// Raising errors to caller for more flexible handling
				return err
			}

			if !predicate(&plugin) {
				continue
			}

			foundKey = _b8toi(k)
			foundValue = &plugin
			return nil
		}

		return nil
	})
	if err != nil {
		return -1, nil, err
	}

	return foundKey, foundValue, nil
}

func (ip *InstalledPlugins) Insert(value *PluginInfo) (int, error) {
	content, err := json.Marshal(value)
	if err != nil {
		return -1, err
	}

	var insertedKey int = -1
	err = ip.boltdb.Update(func(tx *bolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists([]byte(_PluginListBucketName))
		if err != nil {
			return err
		}

		// Generate auto-incremental ID for the plugin information.
		// According to the documentation of https://github.com/etcd-io/bbolt,
        // the NextSequence() method returns an error only if the Tx is closed
		// or not writeable. That can't happen in an Update() call, so the error
		// check can be safely ignored.
        id, _ := bucket.NextSequence()
		insertedKey = int(id)

		return bucket.Put(_itob(insertedKey), []byte(content))
	})
	if err != nil {
		return -1, err
	}

	return insertedKey, nil
}

// Update method simply stores new value to specified position in JSON array.
// Would PANIC if key is out of range.
func (ip *InstalledPlugins) Update(key int, value *PluginInfo) error {
	content, err := json.Marshal(value)
	if err != nil {
		return err
	}

	return ip.boltdb.Update(func(tx *bolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists([]byte(_PluginListBucketName))
		if err != nil {
			return err
		}

		return bucket.Put(_itob(key), []byte(content))
	})
}

func (ip *InstalledPlugins) DeleteByKey(key int) error {
	return ip.boltdb.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(_PluginListBucketName))
		if bucket == nil {
			return nil
		}

		return bucket.Delete(_itob(key))
	})
}

// _itob returns an 8-byte little endian representation of v.
func _itob(v int) []byte {
    b := make([]byte, 8)
    binary.LittleEndian.PutUint64(b, uint64(v))
    return b
}

func _b8toi(b []byte) int {
	return int(binary.LittleEndian.Uint64(b))
}
