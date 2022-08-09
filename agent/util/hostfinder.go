package util

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"sync"

	"github.com/jarcoal/httpmock"

	"github.com/aliyun/aliyun_assist_client/agent/log"
	"github.com/aliyun/aliyun_assist_client/agent/util/networkcategory"
)

const (
	HYBRID_DOMAIN     = ".axt.aliyuncs.com"
	HYBRID_DOMAIN_VPC = ".axt.aliyun.com"
)

var region_ids []string = []string{
	"cn-hangzhou",
	"cn-qingdao",
	"cn-beijing",
	"cn-zhangjiakou",
	"cn-huhehaote",
	"cn-shanghai",
	"cn-shenzhen",
	"cn-hongkong",
	"eu-west-1"}

var (
	g_regionId           string = ""
	g_domainId                  = ""
	g_azoneId                   = ""
	g_instanceId                = ""
	g_regionIdInitLock   sync.Mutex
	g_domainIdInitLock   sync.Mutex
	g_azoneIdInitLock    sync.Mutex
	g_instanceIdInitLock sync.Mutex
)

func connectionDetect(regionId string) error {
	host := regionId + ".axt.aliyun.com"
	url := "https://" + host + "/luban/api/connection_detect"
	err, ret := HttpGet(url)
	if err != nil {
		return err
	}

	if ret == "ok" {
		return nil
	}

	return errors.New("Unknown response")
}

func getRegionIdInVpc() (error, string) {
	url := "http://100.100.100.200/latest/meta-data/region-id"
	err, regionId := HttpGet(url)
	if err != nil {
		return err, ""
	}
	err = connectionDetect(regionId)
	if err == nil {
		return nil, regionId
	}
	return err, ""
}

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

func checkClassicRegion(possibleRegionID string) (string, error) {
	host := possibleRegionID + ".axt.aliyun.com"
	url := "https://" + host + "/luban/api/classic/region-id"
	err, detectedRegionID := HttpGet(url)
	if err != nil {
		return "", err
	}

	// Response of https://available-region.axt.aliyun.com/luban/api/classic/region-id
	// in classic network should be correct region id information for requesting
	// instance, so just check connectivity of responded-region.axt.aliyun.com
	if connectionDetect(detectedRegionID) == nil {
		return detectedRegionID, nil
	}
	return "", errors.New("Unknown regionId")
}

func pollingRegionId() string {
	for _, possibleRegionID := range region_ids {
		detectedRegionID, err := checkClassicRegion(possibleRegionID)
		if err == nil {
			return detectedRegionID
		}
	}
	return ""
}

func getRegionIdInHybrid() string {
	path, _ := GetHybridPath()
	path += "/region-id"
	if CheckFileIsExist(path) {
		raw, err := ioutil.ReadFile(path)
		if err == nil {
			content := string(raw)
			content = strings.Trim(content, "\r")
			content = strings.Trim(content, "\n")
			content = strings.Trim(content, "\t")
			return strings.TrimSpace(content)
		}
	}
	return ""
}

func getNetworkTypeInHybrid() string {
	path, _ := GetHybridPath()
	path += "/network-mode"
	if CheckFileIsExist(path) {
		raw, err := ioutil.ReadFile(path)
		if err == nil {
			content := string(raw)
			content = strings.Trim(content, "\r")
			content = strings.Trim(content, "\n")
			content = strings.Trim(content, "\t")
			return strings.TrimSpace(content)
		}
	}
	return ""
}

func IsHybrid() bool {
	path, _ := GetHybridPath()
	path += "/instance-id"
	if CheckFileIsExist(path) {
		return true
	} else {
		return false
	}
}

func IsSelfHosted() bool {
	return os.Getenv("ALIYUN_ASSIST_SERVER_HOST") != ""
}

func getRegionIdInFile() string {
	cur, _ := GetCurrentPath()
	path := cur + "../region-id"
	if CheckFileIsExist(path) == false {
		return ""
	}
	raw, err := ioutil.ReadFile(path)
	if err == nil {
		content := string(raw)
		content = strings.Trim(content, "\r")
		content = strings.Trim(content, "\n")
		content = strings.Trim(content, "\t")
		return strings.TrimSpace(content)
	}
	return ""
}

// InitRegionId detects and sets region id of server host in both VPC and classic networks
func initRegionId() error {
	if g_regionId == "" {
		var err error
		regionId := getRegionIdInHybrid()
		if regionId != "" {
			g_regionId = regionId

			// Since region id is determined via hybrid cloud-related function,
			// network category is set to NetworkHybrid
			networkcategory.Set(networkcategory.NetworkHybrid)

			return nil
		}
		// Retrieve region ID from meta server in VPC network
		err, regionId = getRegionIdInVpc()
		if err == nil {
			g_regionId = regionId

			// Since region id is determined via VPC-related functions, network
			// category is set to NetworkVPC
			networkcategory.Set(networkcategory.NetworkVPC)

			return nil
		}

		g_regionId = getRegionIdInFile()
		if g_regionId != "" {
			return nil
		}
		// Else, retrieve region ID by polling preset servers
		log.GetLogger().Infoln("Poll region id for instance in classic network")
		g_regionId = pollingRegionId()

		// Since region id is determined via classic network-related function,
		// network category is set to NetworkClassic
		networkcategory.Set(networkcategory.NetworkClassic)
	}
	return nil
}

func getDomainbyMetaServer() string {
	url := "http://100.100.100.200/latest/global-config/aliyun-assist-server-url"
	err, domain := HttpGet(url)

	if err != nil {
		return ""
	}

	g_domainIdInitLock.Lock()
	defer g_domainIdInitLock.Unlock()

	if strings.Contains(domain, "https://") {
		url = domain + "/luban/api/connection_detect"
		err, _ = HttpGet(url)
		if err != nil {
			return ""
		}

		g_domainId := domain[8:]

		// Since axt server domain is directly retrieved from metaserver, network
		// category is set to NetworkWithMetaserver
		networkcategory.Set(networkcategory.NetworkWithMetaserver)

		return g_domainId
	}

	return ""
}

// GetRegionId should be always successful since InitRegionId should be called before.
// However, it's hard to terminate the agent under this case since even panic could
// only terminate the calling goroutine
func GetRegionId() string {
	g_regionIdInitLock.Lock()
	defer g_regionIdInitLock.Unlock()

	if g_regionId != "" {
		return g_regionId
	} else {
		initRegionId()
	}
	return g_regionId
}

// GetServerHost returns empty string when region id is invalid as error handling
func GetServerHost() string {
	g_domainIdInitLock.Lock()
	defer g_domainIdInitLock.Unlock()
	if g_domainId != "" {
		return g_domainId
	}
	if IsSelfHosted() {
		return os.Getenv("ALIYUN_ASSIST_SERVER_HOST")
	}
	regionId := GetRegionId()
	if regionId != "" {
		if IsHybrid() {
			if getNetworkTypeInHybrid() == "vpc" {
				return regionId + HYBRID_DOMAIN_VPC
			} else {
				return regionId + HYBRID_DOMAIN
			}
		}
		return regionId + ".axt.aliyun.com"
	} else {
		return ""
	}

}

func GetDeamonUrl() string {
	url := "https://" + GetServerHost() + "/luban/api/assist_deamon"
	return url
}

func MockMetaServer(region_id string) {
	httpmock.RegisterResponder("GET", "http://100.100.100.200/latest/meta-data/region-id",
		httpmock.NewStringResponder(200, region_id))
	httpmock.RegisterResponder("GET", fmt.Sprintf("https://%s.axt.aliyun.com/luban/api/connection_detect", region_id),
		httpmock.NewStringResponder(200, "ok"))
}
