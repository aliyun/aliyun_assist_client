package commandermanager

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/aliyun/aliyun_assist_client/agent/log"
	"github.com/aliyun/aliyun_assist_client/agent/pluginmanager"
	"github.com/aliyun/aliyun_assist_client/agent/pluginmanager/acspluginmanager"
	"github.com/aliyun/aliyun_assist_client/agent/taskengine/taskerrors"
	"github.com/aliyun/aliyun_assist_client/common/fileutil"
	"github.com/aliyun/aliyun_assist_client/common/pathutil"
	"github.com/aliyun/aliyun_assist_client/interprocess/messagebus/buses"
	"github.com/aliyun/aliyun_assist_client/thirdparty/sirupsen/logrus"
	"github.com/google/uuid"
)

type CommanderManager struct {
	// name -> commander
	commanders map[string]*Commander
	// name -> version
	needUpdate map[string]string

	// name -> handshaketoken
	handshakeToken sync.Map

	l sync.Mutex
}

const (
	ContainerCommanderName = "ACS-ECS-ContainerCommander"

	AgentApiVersion = "v1"
)

var _cm *CommanderManager

// InitCommanderManager init commander manager and load Commander which names
// commanderName from installed_plugins file, all local Commanders will be
// loaded if commanderName is empty.
func InitCommanderManager(commanderName string) {
	_cm = &CommanderManager{}
	_cm.commanders = make(map[string]*Commander)
	_cm.needUpdate = make(map[string]string)

	var err error
	acspluginmanager.PLUGINDIR, err = pathutil.GetPluginPath()
	if err != nil {
		log.GetLogger().WithError(err).Error("InitCommanderManager: get PLUGINDIR failed.")
		return
	}
	_cm.loadCommanderFromLocal(commanderName)
	pluginmanager.SetUpdateHandler(func(name, version string) bool {
		return _cm.markUpdate(name, version)
	})
}

// GetCommanderRunning return the named Commander, Commander will be started
// before return.
func GetCommanderRunning(name string) (*Commander, *taskerrors.CommanderError) {
	return _cm.getCommanderRunning(name)
}

// GetCommander return the named Commander.
func GetCommander(name string) (*Commander, *taskerrors.CommanderError) {
	return _cm.getCommander(name)
}

// GetCommanderWhenHandshake will be called during the handshake, it finds
// the named Commander and check handshake token.
func GetCommanderWhenHandshake(name, token string) (*Commander, error) {
	return _cm.getCommanderWhenHandshake(name, token)
}

func (m *CommanderManager) loadCommanderFromLocal(name string) error {
	commanders, err := acspluginmanager.LoadAllPluginFromLocal(pluginmanager.PLUGIN_COMMANDER)
	if err != nil {
		return err
	}
	if len(commanders) == 0 {
		log.GetLogger().Warn("no commander found")
		if name != "" {
			return fmt.Errorf("not found")
		}
		return nil
	}
	var found bool
	for _, c := range commanders {
		if name != "" && c.Name != name {
			continue
		}
		cmdPath := filepath.Join(acspluginmanager.PLUGINDIR, c.Name, c.Version, c.RunPath)
		if !fileutil.CheckFileIsExist(cmdPath) {
			log.GetLogger().Errorf("Commander %s found in local but the cmdPath[%s] not exist", c.Name, cmdPath)
			continue
		}
		endpoint := buses.NewEndpoint(c.CommanderInfo.EndpointType, filepath.Join(os.TempDir(), c.CommanderInfo.EndpointFile))
		pidFile := filepath.Join(os.TempDir(), c.CommanderInfo.PidFile)
		cfg := &CommanderConfig{
			CommanderName: c.Name,
			CmdPath:       cmdPath,
			PidFile:       pidFile,
			Endpoint:      endpoint,
			Version:       c.Version,
			ApiVersion:    c.CommanderInfo.ApiVersion,

			// default timeout
			StartTimeout:  time.Duration(5) * time.Second,
			AttachTimeout: time.Duration(5) * time.Second,
			DialTimeout:   time.Duration(5) * time.Second,
		}
		commander := NewCommander(cfg)
		m.commanders[c.Name] = commander
		found = true
		log.GetLogger().WithFields(logrus.Fields{
			"name":    c.Name,
			"version": c.Version,
		}).Info("commander loaded")
	}
	if name != "" && !found {
		return fmt.Errorf("not found") 
	}
	return nil
}

func (m *CommanderManager) installCommanderFromOnline(name, version string) error {
	pluginInfo, err := acspluginmanager.QueryPluginFromOnline(name, pluginmanager.PLUGIN_COMMANDER, version)
	if err != nil {
		log.GetLogger().WithFields(logrus.Fields{
			"name":    name,
			"version": version,
		}).WithError(err).Error("query commander from online failed")
		return err
	}
	// TODO: timeout for installing plugin from online
	if err := acspluginmanager.InstallPluginFromOnline(pluginInfo, 30); err != nil {
		log.GetLogger().WithFields(logrus.Fields{
			"name":    name,
			"version": version,
		}).WithError(err).Error("install commander from online failed")
		return err
	}
	log.GetLogger().WithFields(logrus.Fields{
		"name":    pluginInfo.Name,
		"version": pluginInfo.Version,
	}).Info("commander installed from online")
	return nil
}

func (m *CommanderManager) getCommanderRunning(name string) (*Commander, *taskerrors.CommanderError) {
	m.l.Lock()
	defer m.l.Unlock()

	// 1. If commander is loaded and is running, return it.
	if c, ok := m.commanders[name]; ok && c.IsRunning() {
		return c, nil
	}

	// Generate handshake token
	handshakeToken := uuid.NewString()
	m.handshakeToken.Store(name, handshakeToken)
	defer m.handshakeToken.Delete(name)

	// 2. If commander need update install new version from online,
	// else start the loaded commander and return it.
	errMsg := []string{}
	if v, ok := m.needUpdate[name]; ok {
		log.GetLogger().Infof("try to update commander[%s] to version[%s]", name, v)
		if err := m.installCommanderFromOnline(name, v); err != nil {
			errMsg = append(errMsg, fmt.Sprintf("install %s %s from online, %v", name, v, err))
		}
	} else if c, ok := m.commanders[name]; ok {
		if err := c.Start(handshakeToken); err == nil {
			log.GetLogger().Info("start commander success")
			return c, nil
		} else {
			log.GetLogger().Error("start commander failed: ", err)
		}
	}

	// 3. Try to load commander from local or install from online,
	// then start commander and return it.
	if err := m.loadCommanderFromLocal(name); err != nil {
		errMsg = append(errMsg, fmt.Sprintf("load %s from local, %v", name, err))
		if err := m.installCommanderFromOnline(name, ""); err != nil {
			errMsg = append(errMsg, fmt.Sprintf("install %s from online, %v", name, err))
			return nil, taskerrors.NewLoadCommanderFailedError(errMsg...)
		}
		if err := m.loadCommanderFromLocal(name); err != nil {
			errMsg = append(errMsg, fmt.Sprintf("load %s from local, %v", name, err))
		}
	}
	if c, ok := m.commanders[name]; !ok {
		errMsg = append(errMsg, fmt.Sprintf("%s not found", name))
		return nil, taskerrors.NewLoadCommanderFailedError(errMsg...)
	} else {
		return c, c.Start(handshakeToken)
	}
}

func (m *CommanderManager) getCommander(name string) (*Commander, *taskerrors.CommanderError) {
	m.l.Lock()
	defer m.l.Unlock()

	//  1. If commander is loaded, return it.
	if c, ok := m.commanders[name]; ok {
		return c, nil
	}

	// 2. Install commander from online.
	errMsg := []string{}
	if err := m.installCommanderFromOnline(name, ""); err != nil {
		errMsg = append(errMsg, fmt.Sprintf("install %s from online, %v", name, err))
	}

	// 3. Try to load commander from local and return it.
	if err := m.loadCommanderFromLocal(name); err != nil {
		errMsg = append(errMsg, fmt.Sprintf("load %s from local, %v", name, err))
	}
	if c, ok := m.commanders[name]; !ok {
		errMsg = append(errMsg, fmt.Sprintf("%s not found", name))
		return nil, taskerrors.NewLoadCommanderFailedError(errMsg...)
	} else {
		return c, nil
	}
}

func (m *CommanderManager) getCommanderWhenHandshake(name, token string) (*Commander, error) {
	if t, ok := m.handshakeToken.Load(name); ok {
		if tt, ok := t.(string); ok && tt == token {
			if c, ok := m.commanders[name]; ok {
				return c, nil
			} else {
				return nil, fmt.Errorf("commander not found")
			}
		} else {
			return nil, fmt.Errorf("commander's handshake not match")
		}
	}
	return nil, fmt.Errorf("commander's handshake not found")
}

func (m *CommanderManager) markUpdate(name, version string) bool {
	m.l.Lock()
	defer m.l.Unlock()
	var c *Commander
	var ok bool
	if c, ok = m.commanders[name]; !ok {
		return false
	}
	if c.config.Version == version {
		log.GetLogger().Infof("commander[%s] version is already %s, no need to update", name, version)
	} else {
		log.GetLogger().Infof("mark commander[%s] need to update from %s to %s", name, c.config.Version, version)
		m.needUpdate[name] = version
	}
	return true
}
