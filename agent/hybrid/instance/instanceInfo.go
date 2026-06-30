package instance

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/aliyun/aliyun_assist_client/common/fileutil"
	"github.com/aliyun/aliyun_assist_client/common/pathutil"
)

const (
	instanceIdFile             = "instance-id"
	machineIdSourceFilename    = "machine-id-source"
)

func SaveInstanceInfo(instanceId, fingerprint, regionId, pubKey, priKey, networkMode, fpSource string) {
	path, _ := pathutil.GetHybridPath()
	fileutil.WriteAndSyncFile(filepath.Join(path, "instance-id"), []byte(instanceId), 0600)
	fileutil.WriteAndSyncFile(filepath.Join(path, "region-id"), []byte(regionId), 0600)
	fileutil.WriteAndSyncFile(filepath.Join(path, "pub-key"), []byte(pubKey), 0600)
	fileutil.WriteAndSyncFile(filepath.Join(path, "pri-key"), []byte(priKey), 0600)
	fileutil.WriteAndSyncFile(filepath.Join(path, "network-mode"), []byte(networkMode), 0600)
	// Store fingerprint into hybrid/machine-id
	fileutil.WriteAndSyncFile(filepath.Join(path, "machine-id"), []byte(fingerprint), 0600)

	fileutil.WriteAndSyncFile(filepath.Join(path, machineIdSourceFilename), []byte(fpSource), 0600)

	// "Overwrite" the obsolete fingerprint file when persisting new
	// registration information of hybrid instance, with just deletion.
	DeleteDiscardFingerprintFile()
}

func SaveFingerprint(fingerprint string) error {
	path, _ := pathutil.GetHybridPath()
	return fileutil.WriteAndSyncFile(filepath.Join(path, "machine-id"), []byte(fingerprint), 0600)
}

func ReadInstanceId() string {
	path, _ := pathutil.GetHybridPath(pathutil.GPNoCreation)
	instanceId, _ := os.ReadFile(filepath.Join(path, "instance-id"))
	return strings.TrimSpace(string(instanceId))
}

func ReadFingerprint() string {
	path, _ := pathutil.GetHybridPath(pathutil.GPNoCreation)
	fingerprint, _ := os.ReadFile(filepath.Join(path, "machine-id"))
	return strings.TrimSpace(string(fingerprint))
}

// read from fingerprint file which is discard
func ReadDiscardFingerprintFile() string {
	path, _ := pathutil.GetHybridPath(pathutil.GPNoCreation)
	discardFingerprintPath := filepath.Join(path, "fingerprint")
	if _, err := os.Stat(discardFingerprintPath); os.IsNotExist(err) {
		return ""
	}
	machineId, _ := os.ReadFile(discardFingerprintPath)
	return strings.TrimSpace(string(machineId))
}

// delete from fingerprint file which is discard
func DeleteDiscardFingerprintFile() error {
	path, err := pathutil.GetHybridPath(pathutil.GPNoCreation)
	if err != nil {
		return err
	}
	return os.Remove(filepath.Join(path, "fingerprint"))
}

func ReadRegionId() string {
	path, _ := pathutil.GetHybridPath(pathutil.GPNoCreation)
	regionId, _ := os.ReadFile(filepath.Join(path, "region-id"))
	return strings.TrimSpace(string(regionId))
}

func ReadNetworkMode() string {
	path, _ := pathutil.GetHybridPath(pathutil.GPNoCreation)
	networkMode, _ := os.ReadFile(filepath.Join(path, "network-mode"))
	return strings.TrimSpace(string(networkMode))
}

func ReadPriKey() string {
	path, _ := pathutil.GetHybridPath(pathutil.GPNoCreation)
	priKey, _ := os.ReadFile(filepath.Join(path, "pri-key"))
	return strings.TrimSpace(string(priKey))
}

func ReadPubKey() string {
	path, _ := pathutil.GetHybridPath(pathutil.GPNoCreation)
	pubKey, _ := os.ReadFile(filepath.Join(path, "pub-key"))
	return strings.TrimSpace(string(pubKey))
}

func ReadMachineIdSource() string {
	path, err := pathutil.GetHybridPath(pathutil.GPNoCreation)
	if err != nil {
		return ""
	}

	machineIdSource, err := os.ReadFile(filepath.Join(path, machineIdSourceFilename))
	if err != nil {
		return ""
	}

	return strings.TrimSpace(string(machineIdSource))
}

func RemoveInstanceInfo() {
	path, _ := pathutil.GetHybridPath(pathutil.GPNoCreation)
	os.Remove(filepath.Join(path, "instance-id"))
	os.Remove(filepath.Join(path, "machine-id"))
	os.Remove(filepath.Join(path, "region-id"))
	os.Remove(filepath.Join(path, "pub-key"))
	os.Remove(filepath.Join(path, "pri-key"))
	os.Remove(filepath.Join(path, "network-mode"))

	os.Remove(filepath.Join(path, machineIdSourceFilename))

	// Garbage-collect the obsolete fingerprint file by deletion, if the
	// migration on agent startup has not been done.
	DeleteDiscardFingerprintFile()
}

func IsHybrid() bool {
	hybridDir, _ := pathutil.GetHybridPath(pathutil.GPNoCreation)
	path := filepath.Join(hybridDir, "instance-id")

	_, err := os.Stat(path)
	return !errors.Is(err, os.ErrNotExist)
}
