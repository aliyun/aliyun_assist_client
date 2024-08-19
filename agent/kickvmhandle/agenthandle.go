package kickvmhandle

import (
	"errors"
	"fmt"
	"strings"

	"github.com/aliyun/aliyun_assist_client/agent/log"
	"github.com/aliyun/aliyun_assist_client/agent/update"
	"github.com/aliyun/aliyun_assist_client/agent/util/versionutil"
	"github.com/aliyun/aliyun_assist_client/agent/version"
)

// type:agent
var agentRoute map[string]handleFunc

func init() {
	agentRoute = map[string]handleFunc{
		"stop":   stopAgant,
		"remove": removeAgant,
		"update": updateAgant,

		"rollback": rollbackAgent,
		"upgrade":  upgradeAgent,
	}
}

type AgentHandle struct {
	action string
	params []string
}

func NewAgentHandle(action string, params []string) *AgentHandle {
	return &AgentHandle{
		action: action,
		params: params,
	}
}

func (h *AgentHandle) DoAction() error {
	if v, ok := agentRoute[h.action]; ok {
		v(h.params)
	} else {
		return errors.New("no action found")
	}
	return nil
}

func (h *AgentHandle) CheckAction() bool {
	if _, ok := agentRoute[h.action]; ok {
		return true
	} else {
		return false
	}
}

func rollbackAgent(params []string) error {
	go func() {
		// kick_vm agent rollback <version>
		if len(params) < 1 {
			log.GetLogger().Errorln("params invalid", params)
			return
		}
		newVersion := strings.TrimSpace(params[0])
		if res, err := compareWithCurVersion(newVersion); err != nil {
			log.GetLogger().Error("Kick rollback, compare with current version failed: ", err)
		} else if res >= 0 {
			log.GetLogger().Error("Kick rollback need a lower version")
		} else {
			if err := update.RollbackWithLocalDir(newVersion, update.UpdateReasonKickRollback); err != nil {
				log.GetLogger().Error("Kick rollback failed: ", err)
			}
		}
	}()
	return nil
}

func upgradeAgent(params []string) error {
	go func() {
		// kick_vm agent upgrade <version>
		if len(params) < 1 {
			log.GetLogger().Errorln("params invalid", params)
			return
		}
		newVersion := strings.TrimSpace(params[0])
		if res, err := compareWithCurVersion(newVersion); err != nil {
			log.GetLogger().Error("Kick upgrade, compare with current version failed: ", err)
		} else if res <= 0 {
			log.GetLogger().Error("Kick upgrade need a higher version")
		} else {
			if err := update.UpgradeWithLocalDir(newVersion, update.UpdateReasonKickupgrade); err != nil {
				log.GetLogger().WithError(err).Error("Kick upgrade failed, try to trigger update check")
				if !errors.Is(err, update.ErrUpdateIsDisabled) {
					update.TriggerUpdateCheck()
				}
			}
		}
	}()
	return nil
}

// compareWithCurVersion compare newVersion with current version. The version
// number format is `a.b.c.d`. If the `a.b` part of the two version numbers does
// not match, an error will be returned. We just compare the `c.d` part.
func compareWithCurVersion(newVersion string) (int, error) {
	newVersionParts := strings.Split(newVersion, ".")
	if len(newVersionParts) != 4 {
		return 0, fmt.Errorf("new version format invalid")
	}
	versionParts := strings.Split(version.AssistVersion, ".")
	if len(versionParts) != 4 {
		return 0, fmt.Errorf("current version format invalid")
	}
	if newVersionParts[0] != versionParts[0] || newVersionParts[1] != versionParts[1] {
		return 0, fmt.Errorf("main fields not match")
	}
	return versionutil.CompareVersion(newVersion, version.AssistVersion), nil
}
