package osutil

import (
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/aliyun/aliyun_assist_client/common/fileutil"
	"github.com/stretchr/testify/assert"
)

func TestGetDistribution(t *testing.T) {
	var mockCheckfileExist bool
	defer gomonkey.ApplyFunc(fileutil.CheckFileIsExist, func(fileName string) bool {
		return mockCheckfileExist
	}).Reset()

	mockCheckfileExist = false
	distribution := GetDistribution()
	assert.Equal(t, "", distribution)

	mockCheckfileExist = true
	distribution = GetDistribution()
	assert.Equal(t, distributionAndroid, distribution)
}
