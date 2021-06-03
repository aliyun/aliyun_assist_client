// +build darwin freebsd linux netbsd openbsd

package osutil

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetPlatformName(t *testing.T) {
	val, err := getPlatformName()
	if err != nil {
		assert.Contains(t, err.Error(), "")
	} else {
		assert.NotNil(t, val)
	}
}

func TestGetPlatformType(t *testing.T) {
	val, _ := getPlatformType()
	assert.Equal(t, "linux", val)
}