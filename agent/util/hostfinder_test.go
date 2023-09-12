package util

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/aliyun/aliyun_assist_client/common/pathutil"
)

func TestGetRegionIdInFile(t *testing.T) {
	cur, _ := pathutil.GetCurrentPath()
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
	path,_ := pathutil.GetHybridPath()
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
