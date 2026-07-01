package instance

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInstanceInfo(t *testing.T) {
	instanceId := "i-xxx"
	fingerprint := "000_38B133FFCA7749598D4274E36EBE932E"
	regionId := "the-region"
	pubKey := "the-pubKey"
	priKey := "the-priKey"
	networkMode := "vpc"
	machineIdSource := "the-source"

	SaveInstanceInfo(instanceId, fingerprint, regionId, pubKey, priKey, networkMode, machineIdSource)
	assert.Equal(t, instanceId, ReadInstanceId())
	assert.Equal(t, fingerprint, ReadFingerprint())
	assert.Equal(t, regionId, ReadRegionId())
	assert.Equal(t, pubKey, ReadPubKey())
	assert.Equal(t, priKey, ReadPriKey())
	assert.Equal(t, networkMode, ReadNetworkMode())
	assert.Equal(t, machineIdSource, ReadMachineIdSource())

	RemoveInstanceInfo()
	assert.Equal(t, "", ReadInstanceId())
}
