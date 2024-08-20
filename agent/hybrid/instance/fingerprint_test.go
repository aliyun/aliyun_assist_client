package instance

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	invalidUTF8String = "\xbd\xb2\x3d\xbc\x20\xe2\x8c\x98"
	ipAddressID       = "ipaddress-info"
)

func TestInstanceFingerprint(t *testing.T) {
	RemoveHardwareInfo()
	fingerprint1, err := GenerateFingerprint()
	assert.Equal(t, nil, err)

	fingerprint2, err := GenerateFingerprint()
	assert.Equal(t, nil, err)
	assert.Equal(t, fingerprint1, fingerprint2)

	fingerprint2, err = GenerateFingerprint()
	assert.Equal(t, nil, err)
	assert.Equal(t, fingerprint1, fingerprint2)
}

func TestIsValidHardwareHash_ReturnsHashIsValid(t *testing.T) {
	// Arrange
	sampleHash := make(map[string]string)
	sampleHash[hardwareID] = "sample"

	// Act
	isValid := isValidHardwareHash(sampleHash)

	// Assert
	assert.True(t, isValid)
}

func TestIsValidHardwareHash_ReturnsHashIsInvalid(t *testing.T) {
	// Arrange
	sampleHash := make(map[string]string)
	sampleHash[hardwareID] = invalidUTF8String

	//Act
	isValid := isValidHardwareHash(sampleHash)

	// Assert
	assert.False(t, isValid)
}

func TestIsSimilarHardwareHash_Negtive(t *testing.T) {
	origin := map[string]string{
		hardwareID:      "hardwareValue",
		ipAddressID:     "ipAddressValue",
		"somethingElse": "A",
	}

	current := map[string]string{
		hardwareID:      "hardwareValue",
		ipAddressID:     "ipAddressValue_Other",
		"somethingElse": "A",
	}

	ret := isSimilarHardwareHash(origin, current, 80)
	assert.Equal(t, ret, false)
}

func TestIsSimilarHardwareHash_Positive(t *testing.T) {
	origin := map[string]string{
		hardwareID:      "hardwareValue",
		ipAddressID:     "ipAddressValue",
		"somethingElse": "A",
	}

	current := map[string]string{
		hardwareID:      "hardwareValue",
		ipAddressID:     "ipAddressValue",
		"somethingElse": "B",
	}

	ret := isSimilarHardwareHash(origin, current, 50)
	assert.Equal(t, ret, true)
}

func TestGenerateUUID(t *testing.T) {
	uuid := generateUUID()
	fmt.Println(uuid)
	assert.True(t, len(uuid) <= 36)
	assert.True(t, strings.HasPrefix(uuid, fingerprintVersionPrefix))
}
