package hwid1

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/aliyun/aliyun_assist_client/common/fileutil"
	"github.com/aliyun/aliyun_assist_client/common/pathutil"
)

type HardwareInfo struct {
	Fingerprint  string
	HardwareHash map[string]string
}

const (
	defaultMatchPercent = 40
	hwInfoFile          = "hardwareHash"
	RWPermission   os.FileMode = 0600

	ipaddreddInfo = "ipaddress-info"
)

func SaveHardwareInfo(fingerprint string, hardwareHash map[string]string) error {
	hardwareInfo := &HardwareInfo{
		Fingerprint:  fingerprint,
		HardwareHash: hardwareHash,
	}
	content, err := json.Marshal(hardwareInfo)
	if err != nil {
		return err
	}

	path, _ := pathutil.GetHybridPath()
	hwInfoPath := filepath.Join(path, hwInfoFile)
	return fileutil.WriteAndSyncFile(hwInfoPath, content, os.ModePerm & RWPermission)
}

func ReadHardwareInfo() (*HardwareInfo, error) {
	path, _ := pathutil.GetHybridPath(pathutil.GPNoCreation)
	hwInfoPath := filepath.Join(path, hwInfoFile)
	content, err := os.ReadFile(hwInfoPath)
	if err != nil {
		return nil, err
	}
	hwInfo := &HardwareInfo{}
	err = json.Unmarshal(content, hwInfo)
	if err != nil {
		return nil, err
	}
	return hwInfo, nil
}

func RemoveHardwareInfo() {
	path, _ := pathutil.GetHybridPath(pathutil.GPNoCreation)
	hwInfoPath := filepath.Join(path, hwInfoFile)
	os.Remove(hwInfoPath)
}
