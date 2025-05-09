package util

import (
	"os"
	"sync"

	"github.com/aliyun/aliyun_assist_client/agent/hybrid/instance"
	"github.com/aliyun/aliyun_assist_client/agent/log"
	"github.com/aliyun/aliyun_assist_client/common/metaserver"
	"github.com/aliyun/aliyun_assist_client/common/requester"
)

const (
	// HYBRID_DOMAIN_VPC is same as apiserver.IntranetDomain
	HYBRID_DOMAIN_VPC = ".axt.aliyun.com"
)

var (
	g_domainId           = ""
	g_azoneId            = ""
	g_instanceId         = ""
	g_domainIdInitLock   sync.RWMutex
	g_azoneIdInitLock    sync.RWMutex
	g_instanceIdInitLock sync.RWMutex
)

func GetAzoneId() string {
	g_azoneIdInitLock.RLock()
	if g_azoneId == "" {
		g_azoneIdInitLock.RUnlock()
		g_azoneIdInitLock.Lock()
		defer g_azoneIdInitLock.Unlock()
		if g_azoneId == "" {
			if azoneId, err := metaserver.GetZoneId(log.GetLogger()); err != nil {
				g_azoneId = "unknown"
			} else {
				g_azoneId = azoneId
			}
		}
	} else {
		defer g_azoneIdInitLock.RUnlock()
	}
	return g_azoneId
}

func GetInstanceId() string {
	g_instanceIdInitLock.RLock()
	if g_instanceId == "" {
		g_instanceIdInitLock.RUnlock()
		g_instanceIdInitLock.Lock()
		defer g_instanceIdInitLock.Unlock()
		if g_instanceId == "" {
			if instance.IsHybrid() {
				if instanceId := instance.ReadInstanceId(); instanceId != "" {
					g_instanceId = instanceId
				} else {
					g_instanceId = "unknown"
				}
			} else {
				if instanceId, err := metaserver.GetInstanceId(log.GetLogger()); err != nil {
					g_instanceId = "unknown"
				} else {
					g_instanceId = instanceId
				}
			}
		}
	} else {
		defer g_instanceIdInitLock.RUnlock()
	}
	return g_instanceId
}

func IsSelfHosted() bool {
	return os.Getenv("ALIYUN_ASSIST_SERVER_HOST") != ""
}

func GetRegionId() string {
	regionId, err := requester.GetRegionId(log.GetLogger())
	if err != nil {
		log.GetLogger().WithError(err).Errorln("Failed to determine region id")
	}

	return regionId
}

// GetServerHost returns empty string when region id is invalid as error handling
func GetServerHost() string {
	g_domainIdInitLock.RLock()
	if g_domainId == "" {
		g_domainIdInitLock.RUnlock()
		g_domainIdInitLock.Lock()
		defer g_domainIdInitLock.Unlock()
		if g_domainId == "" {
			var err error
			domainId, err := requester.GetServerDomain(log.GetLogger())
			if err != nil {
				log.GetLogger().WithError(err).Errorln("Failed to determine API server domain")
			} else {
				g_domainId = domainId
			}
		}
	} else {
		defer g_domainIdInitLock.RUnlock()
	}
	return g_domainId
}
