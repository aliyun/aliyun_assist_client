package util

import (
	"errors"
	"io/ioutil"
	"os"
	"strings"
	"sync"
)

const (
	HYBRID_DOMAIN = ".axt.aliyuncs.com"
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
	ret, err := HttpGet(url)
	if err != nil {
		return err
	}

	if ret == "ok" {
		return nil
	}

	return errors.New("Unknown response")
}

func getRegionIdInVpc() (error, string) {
	var token, regionId string
	var err error
	url := "http://100.100.100.200/latest/meta-data/region-id"
	token, err = MetaserverToken()
	if err == nil {
		regionId, err = HttpGetWithHeaders(url, map[string]string{
			"X-aliyun-ecs-metadata-token": token,
		})
	} else {
		regionId, err = HttpGet(url)
	}
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
	var token, azoneId string
	var err error
	url := "http://100.100.100.200/latest/meta-data/zone-id"
	token, err = MetaserverToken()
	if err == nil {
		azoneId, err = HttpGetWithHeaders(url, map[string]string{
			"X-aliyun-ecs-metadata-token": token,
		})
	} else {
		azoneId, err = HttpGet(url)
	}
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

	var token, instanceId string
	var err error
	url := "http://100.100.100.200/latest/meta-data/instance-id"
	token, err = MetaserverToken()
	if err == nil {
		instanceId, err = HttpGetWithHeaders(url, map[string]string{
			"X-aliyun-ecs-metadata-token": token,
		})
	} else {
		instanceId, err = HttpGet(url)
	}

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
	detectedRegionID, err := HttpGet(url)
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
		// Retrieve region ID from meta server in VPC network
		err, regionId := getRegionIdInVpc()
		if err == nil {
			g_regionId = regionId
			return nil
		}

		g_regionId = getRegionIdInFile()
		if g_regionId != "" {
			return nil
		}

	}
	return nil
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
		return regionId + ".axt.aliyun.com"
	} else {
		return ""
	}

}
