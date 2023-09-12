package machineid

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/aliyun/aliyun_assist_client/common/pathutil"
)

func GetMachineID() (string, error) {
	hybridDir, _ := pathutil.GetHybridPath()
	path := filepath.Join(hybridDir, "machine-id")
	if machineIdCacheFile, err := os.Open(path); err == nil {
		content, _ := io.ReadAll(machineIdCacheFile)
		cached_id := string(content)
		return cached_id, nil
	}

	return getMachineID()
}

func getMachineID() (string, error) {
	id, err := machineID()
	if err != nil {
		return "", fmt.Errorf("machineid: %v", err)
	}
	return id, nil
}


func protect(appID, id string) string {
	mac := hmac.New(sha256.New, []byte(id))
	mac.Write([]byte(appID))
	return hex.EncodeToString(mac.Sum(nil))
}


func trim(s string) string {
	return strings.TrimSpace(strings.Trim(s, "\n"))
}

func readFile(filename string) ([]byte, error) {
	return ioutil.ReadFile(filename)
}