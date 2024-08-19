package instance

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInstanceInfo(t *testing.T) {
	instanceId := "i-xxx"
	fingerprint, err := GenerateFingerprint()
	assert.Equal(t, nil, err)
	regionId := "the-region"
	pubKey := "the-pubKey"
	priKey := "the-priKey"
	networkMode := "vpc"
	SaveInstanceInfo(instanceId, fingerprint, regionId, pubKey, priKey, networkMode)
	assert.Equal(t, instanceId, ReadInstanceId())
	assert.Equal(t, fingerprint, ReadFingerprint())
	assert.Equal(t, regionId, ReadRegionId())
	assert.Equal(t, pubKey, ReadPubKey())
	assert.Equal(t, priKey, ReadPriKey())
	assert.Equal(t, networkMode, ReadNetworkMode())

	RemoveInstanceInfo()
	assert.Equal(t, "", ReadInstanceId())
}
