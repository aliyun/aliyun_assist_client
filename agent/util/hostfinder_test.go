package util

import (
	"fmt"
	"os"
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
)


func TestGetRegionIdInFile(t *testing.T) {
	cur, _ := GetCurrentPath()
	path := cur + "../region-id"
	defer func(){
		if CheckFileIsExist(path) {
			os.Remove(path)
		}
	}()
	WriteStringToFile(path, "cn-hangzhou\r\n")

	region_id := getRegionIdInFile()
	assert.Equal(t, region_id, "cn-hangzhou")
}

func TestGetRegionIdInHybrid(t *testing.T) {
	path,_ := GetHybridPath()
	regin_path := path + "/region-id"
	instance_path := path + "/instance-id"
	defer func() {
		if CheckFileIsExist(regin_path) {
			os.Remove(regin_path)
		}
		if CheckFileIsExist(instance_path) {
			os.Remove(instance_path)
		}
	}()
	
	WriteStringToFile(instance_path, "cn-hangzhou\r")
	WriteStringToFile(regin_path, "cn-hangzhou\r")

	region_id := getRegionIdInHybrid()
	assert.Equal(t, region_id, "cn-hangzhou")
	is_hybrid := IsHybrid()
	assert.Equal(t, is_hybrid, true)
}

func TestGetDomainByMetaServer(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	NilRequest.Set()
	defer NilRequest.Clear()

	httpmock.RegisterResponder("GET", "http://100.100.100.200/latest/meta-data/region-id",
		httpmock.NewStringResponder(200, `cn-test`))

	httpmock.RegisterResponder("GET", "http://100.100.100.200/latest/global-config/aliyun-assist-server-url",
		httpmock.NewStringResponder(200, `https://abcd.com`))

	httpmock.RegisterResponder("GET", "https://abcd.com/luban/api/connection_detect",
		httpmock.NewStringResponder(200, `true`))

	region_id,_ := getRegionIdInVpc()
	fmt.Println(region_id)

	domain := getDomainbyMetaServer()

	assert.Equal(t, domain, "abcd.com")

}