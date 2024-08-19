package osutil

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPlatformName(t *testing.T) {
	name, err := PlatformName()
	if err != nil {
		assert.Contains(t, err.Error(), "not found")
	} else {
		assert.NotNil(t, name)
	}
}

func TestPlatformVersion(t *testing.T) {
	ver, err := PlatformVersion()
	if err != nil {
		assert.Contains(t, err.Error(), "not found")
	} else {
		assert.NotNil(t, ver)
	}
}


// func TestPlatformArchitect(t *testing.T) {
// 	val, _ := PlatformArchitect()
// 	assert.Equal(t, "x86_64", val)
// }

func TestGetNormalizedPlatform(t *testing.T) {
	val, _ := getNormalizedPlatform("linux")
	assert.Equal(t, "linux", val)
}
