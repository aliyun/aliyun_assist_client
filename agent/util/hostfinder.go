package util

import (
	"os"
	"sync"

	"github.com/aliyun/aliyun_assist_client/agent/log"
	"github.com/aliyun/aliyun_assist_client/common/requester"
)

const (
	HYBRID_DOMAIN     = ".axt.aliyuncs.com"
	HYBRID_DOMAIN_VPC = ".axt.aliyun.com"
)

var (
	g_domainId                  = ""
	g_azoneId                   = ""
	g_instanceId                = ""
	g_domainIdInitLock   sync.Mutex
	g_azoneIdInitLock    sync.Mutex
	g_instanceIdInitLock sync.Mutex
)

func GetAzoneId() string {
	g_azoneIdInitLock.Lock()
	defer g_azoneIdInitLock.Unlock()
	if len(g_azoneId) > 0 {
		return g_azoneId
	}
	url := "http://100.100.100.200/latest/meta-data/zone-id"
	err, azoneId := HttpGet(url)
	if err != nil {
		g_azoneId = "unknown"
		return g_azoneId
	}
	g_azoneId = azoneId
	return g_azoneId
}

func GetInstanceId() string {
	g_instanceIdInitLock.Lock()
	defer g_instanceIdInitLock.Unlock()
	if len(g_instanceId) > 0 {
		return g_instanceId
	}
	url := "http://100.100.100.200/latest/meta-data/instance-id"
	err, instanceId := HttpGet(url)
	if err != nil {
		g_instanceId = "unknown"
		return g_instanceId
	}
	g_instanceId = instanceId
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
	g_domainIdInitLock.Lock()
	defer g_domainIdInitLock.Unlock()
	if g_domainId == "" {
		var err error
		g_domainId, err = requester.GetServerDomain(log.GetLogger())
		if err != nil {
			log.GetLogger().WithError(err).Errorln("Failed to determine API server domain")
		}
	}

	return g_domainId
}
