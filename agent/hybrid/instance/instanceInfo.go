package instance

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"github.com/aliyun/aliyun_assist_client/common/pathutil"
)

const (
	instanceIdFile             = "instance-id"
	RWPermission   os.FileMode = 0600
)

func SaveInstanceInfo(instanceId, fingerprint, regionId, pubKey, priKey, networkMode string) {
	path, _ := pathutil.GetHybridPath()
	os.WriteFile(filepath.Join(path, "instance-id"), []byte(instanceId), 0600)
	os.WriteFile(filepath.Join(path, "region-id"), []byte(regionId), 0600)
	os.WriteFile(filepath.Join(path, "pub-key"), []byte(pubKey), 0600)
	os.WriteFile(filepath.Join(path, "pri-key"), []byte(priKey), 0600)
	os.WriteFile(filepath.Join(path, "network-mode"), []byte(networkMode), 0600)
	// Store fingerprint into hybrid/machine-id
	os.WriteFile(filepath.Join(path, "machine-id"), []byte(fingerprint), 0600)
}

func SaveFingerprint(fingerprint string) error {
	path, _ := pathutil.GetHybridPath()
	return os.WriteFile(filepath.Join(path, "machine-id"), []byte(fingerprint), 0600)
}

func ReadInstanceId() string {
	path, _ := pathutil.GetHybridPath()
	instanceId, _ := os.ReadFile(filepath.Join(path, "instance-id"))
	return strings.TrimSpace(string(instanceId))
}

func ReadFingerprint() string {
	path, _ := pathutil.GetHybridPath()
	fingerprint, _ := os.ReadFile(filepath.Join(path, "machine-id"))
	return strings.TrimSpace(string(fingerprint))
}

// read from fingerprint file which is discard
func ReadDiscardFingerprintFile() string {
	path, _ := pathutil.GetHybridPath()
	discardFingerprintPath := filepath.Join(path, "fingerprint")
	if _, err := os.Stat(discardFingerprintPath); os.IsNotExist(err) {
		return ""
	}
	machineId, _ := os.ReadFile(discardFingerprintPath)
	return strings.TrimSpace(string(machineId))
}

// delete from fingerprint file which is discard
func DeleteDiscardFingerprintFile() error {
	path, err := pathutil.GetHybridPath()
	if err != nil {
		return err
	}
	return os.Remove(filepath.Join(path, "fingerprint"))
}

func ReadRegionId() string {
	path, _ := pathutil.GetHybridPath()
	regionId, _ := os.ReadFile(filepath.Join(path, "region-id"))
	return strings.TrimSpace(string(regionId))
}

func ReadNetworkMode() string {
	path, _ := pathutil.GetHybridPath()
	networkMode, _ := os.ReadFile(filepath.Join(path, "network-mode"))
	return strings.TrimSpace(string(networkMode))
}

func ReadPriKey() string {
	path, _ := pathutil.GetHybridPath()
	priKey, _ := os.ReadFile(filepath.Join(path, "pri-key"))
	return strings.TrimSpace(string(priKey))
}

func ReadPubKey() string {
	path, _ := pathutil.GetHybridPath()
	pubKey, _ := os.ReadFile(filepath.Join(path, "pub-key"))
	return strings.TrimSpace(string(pubKey))
}

func RemoveInstanceInfo() {
	path, _ := pathutil.GetHybridPath()
	os.Remove(filepath.Join(path, "instance-id"))
	os.Remove(filepath.Join(path, "machine-id"))
	os.Remove(filepath.Join(path, "region-id"))
	os.Remove(filepath.Join(path, "pub-key"))
	os.Remove(filepath.Join(path, "pri-key"))
	os.Remove(filepath.Join(path, "network-mode"))
}

func IsHybrid() bool {
	hybridDir, _ := pathutil.GetHybridPath()
	path := filepath.Join(hybridDir, "instance-id")

	_, err := os.Stat(path)
	return !errors.Is(err, os.ErrNotExist)
}
