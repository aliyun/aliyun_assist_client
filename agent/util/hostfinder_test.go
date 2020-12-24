package util

import (
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
	"testing"
)


func TestGetRegionIdInFile(t *testing.T) {
	cur, _ := GetCurrentPath()
	path := cur + "../region-id"
	WriteStringToFile(path, "cn-hangzhou\r\n")

	region_id := getRegionIdInFile()
	assert.Equal(t, region_id, "cn-hangzhou")
}

func TestGetRegionIdInHybrid(t *testing.T) {
	path,_ := GetHybridPath()
	path +=  "/instance-id"
	WriteStringToFile(path, "cn-hangzhou\r")
	WriteStringToFile(path, "cn-hangzhou\r")

	region_id := getRegionIdInHybrid()
	assert.Equal(t, region_id, "cn-hangzhou")

	is_hybrid := IsHybrid()
	assert.Equal(t, is_hybrid, true)
}

func TestGetDomainByMetaServer(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponder("GET", "100.100.100.200/latest/global-config/ecs-assistant-server",
		httpmock.NewStringResponder(200, `https://abcd.com`))

	httpmock.RegisterResponder("GET", "https://abcd.com/luban/api/connection_detect",
		httpmock.NewStringResponder(200, `true`))

	domain := getDomainbyMetaServer()

	assert.Equal(t, domain, "abcd.com")

}