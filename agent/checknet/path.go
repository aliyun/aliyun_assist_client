package checknet

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/aliyun/aliyun_assist_client/agent/log"
	"github.com/aliyun/aliyun_assist_client/agent/util"
	"github.com/aliyun/aliyun_assist_client/thirdparty/sirupsen/logrus"
)

const (
	defaultUnixNetcheckExecutableName = "aliyun_assist_netcheck"
)

var (
	_netcheckPath     string
	_netcheckPathLock sync.Mutex
)

// getNetcheckExecutableName returns netcheck executable name based on OSes
func getNetcheckExecutableName() string {
	return defaultUnixNetcheckExecutableName
}

// initNetcheckPath detects whether netcheck program is bundled within current
// agent release version
func initNetcheckPath() error {
	path, err := os.Executable()
	if err != nil {
		_netcheckPath = ""
		return err
	}

	currentVersionDir, err := filepath.Abs(filepath.Dir(path))
	if err != nil {
		_netcheckPath = ""
		return err
	}

	currentVersionNetcheckPath := filepath.Join(currentVersionDir, getNetcheckExecutableName())
	if !util.CheckFileIsExist(currentVersionNetcheckPath) {
		_netcheckPath = ""
		return fmt.Errorf("Netcheck executable not found at %s", currentVersionNetcheckPath)
	}

	_netcheckPath = currentVersionNetcheckPath
	return nil
}

func getNetcheckPath() string {
	_netcheckPathLock.Lock()
	defer _netcheckPathLock.Unlock()
	if _netcheckPath == "" {
		if err := initNetcheckPath(); err != nil {
			log.GetLogger().WithFields(logrus.Fields{
				"module": "checknet",
			}).WithError(err).Errorln("Failed to detect netcheck executable path")
		}
	}

	return _netcheckPath
}
