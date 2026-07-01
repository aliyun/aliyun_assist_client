package hwid1

import (
	"fmt"
	"strings"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/stretchr/testify/assert"
)

const (
	invalidUTF8String = "\xbd\xb2\x3d\xbc\x20\xe2\x8c\x98"
	ipAddressID       = "ipaddress-info"
)

func TestInstanceFingerprint(t *testing.T) {
	RemoveHardwareInfo()
	fingerprint1, err := (&Fingerprint{}).Generate()
	assert.Equal(t, nil, err)

	fingerprint2, err := (&Fingerprint{}).Generate()
	assert.Equal(t, nil, err)
	assert.Equal(t, fingerprint1, fingerprint2)

	fingerprint2, err = (&Fingerprint{}).Generate()
	assert.Equal(t, nil, err)
	assert.Equal(t, fingerprint1, fingerprint2)
}

func TestCleanup(t *testing.T) {
	t.Run("NormalCase", func(t *testing.T) {
		mockHardwareInfo := &HardwareInfo{
			Fingerprint:  "fake-fingerprint",
			HardwareHash: map[string]string{"key": "value"},
		}
		mockCurrentHash := map[string]string{"new-key": "new-value"}
		var savedFingerprint string
		var savedHash map[string]string
		saveCalled := false

		patches := gomonkey.NewPatches()
		defer patches.Reset()
		patches.ApplyFunc(ReadHardwareInfo, func() (*HardwareInfo, error) {
			return mockHardwareInfo, nil
		})
		patches.ApplyFunc(CurrentHwHash, func() (map[string]string, error) {
			return mockCurrentHash, nil
		})
		patches.ApplyFunc(SaveHardwareInfo, func(fp string, hh map[string]string) error {
			savedFingerprint = fp
			savedHash = hh
			saveCalled = true
			return nil
		})

		err := (&Fingerprint{}).Cleanup()
		assert.NoError(t, err)

		assert.True(t, saveCalled)
		assert.Equal(t, mockHardwareInfo.Fingerprint, savedFingerprint)
		assert.Equal(t, mockCurrentHash, savedHash)
	})

	t.Run("ReadHardwareInfoFailedSilently", func(t *testing.T) {
		saveCalled := false

		patches := gomonkey.NewPatches()
		defer patches.Reset()
		patches.ApplyFunc(ReadHardwareInfo, func() (*HardwareInfo, error) {
			return nil, fmt.Errorf("mock error")
		})
		patches.ApplyFunc(CurrentHwHash, func() (map[string]string, error) {
			return nil, nil
		})
		patches.ApplyFunc(SaveHardwareInfo, func(string, map[string]string) error {
			saveCalled = true
			return nil
		})

		err := (&Fingerprint{}).Cleanup()
		assert.NoError(t, err)
		assert.False(t, saveCalled)
	})

	t.Run("CurrentHwHashFailedSilently", func(t *testing.T) {
		saveCalled := false

		patches := gomonkey.NewPatches()
		defer patches.Reset()
		mockHardwareInfo := &HardwareInfo{Fingerprint: "fake"}
		patches.ApplyFunc(ReadHardwareInfo, func() (*HardwareInfo, error) {
			return mockHardwareInfo, nil
		})
		patches.ApplyFunc(CurrentHwHash, func() (map[string]string, error) {
			return nil, fmt.Errorf("mock error")
		})
		patches.ApplyFunc(SaveHardwareInfo, func(string, map[string]string) error {
			saveCalled = true
			return nil
		})

		err := (&Fingerprint{}).Cleanup()
		assert.NoError(t, err)
		assert.False(t, saveCalled)
	})
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
	assert.True(t, strings.HasPrefix(uuid, FingerprintVersionPrefix))
}
