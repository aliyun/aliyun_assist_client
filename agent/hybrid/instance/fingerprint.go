package instance

import (
	"crypto/md5"
	"encoding/base64"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/aliyun/aliyun_assist_client/agent/log"
	"github.com/google/uuid"
)

const (
	fingerprintVersion = 1
)

var (
	fingerprintVersionPrefix = fmt.Sprintf("%03x_", fingerprintVersion)
)

// GenerateFingerprintIgnoreSavedHash generate fingerprint regardless of whether
//  a hardware hash already exists.
func GenerateFingerprintIgnoreSavedHash() (string, error) {
	return generateFingerprint(true)
}

// GenerateFingerprint generate fingerprint, but if hardware hash already exists 
// and it is similar enough to current hardware the fingerprint in hardware hash
// will be returned.
func GenerateFingerprint() (string, error) {
	return generateFingerprint(false)
}

func generateFingerprint(ignoreSavedHash bool) (string, error) {
	var hardwareHash map[string]string
	var savedHwInfo *HardwareInfo
	var err error
	var hwHashErr error

	// retry getting the new hash and compare with the saved hash for 3 times
	for attempt := 1; attempt <= 3; attempt++ {
		// fetch current hardware hash values
		hardwareHash, hwHashErr = CurrentHwHash()
		if hwHashErr != nil || !isValidHardwareHash(hardwareHash) {
			// sleep 5 seconds until the next retry
			time.Sleep(5 * time.Second)
			continue
		}

		if !ignoreSavedHash {
			// try to get previously saved hardwareInfo
			savedHwInfo, err = ReadHardwareInfo()
			if err != nil {
				log.GetLogger().Errorf("read saved hardwareInfo failed: %s", err)
				continue
			}

			// first time generation, breakout retry
			if !hasFingerprint(savedHwInfo) {
				log.GetLogger().Warn("No fingerprint detected in saved hardwareInfo, skipping retry...")
				break
			}

			// stop retry if the hardware hashes are the same
			if isSimilarHardwareHash(savedHwInfo.HardwareHash, hardwareHash, defaultMatchPercent) {
				log.GetLogger().Info("Calculated hardware hash is same as saved one, returning fingerprint")
				return savedHwInfo.Fingerprint, nil
			}
			log.GetLogger().Warn("Calculated hardware hash is different with saved one, retry to ensure the difference is not cause by the dependency has not been ready")
			// sleep 5 seconds until the next retry
			time.Sleep(5 * time.Second)
		} else {
			log.GetLogger().Info("Ignore saved hardwareInfo")
			break
		}
	}

	if hwHashErr != nil {
		log.GetLogger().Errorf("Error while fetching hardware hashes from instance: %s", hwHashErr)
		return "", hwHashErr
	} else if !isValidHardwareHash(hardwareHash) {
		return "", fmt.Errorf("hardware hash generated contains invalid characters. %s", hardwareHash)
	}
	if ignoreSavedHash {
		log.GetLogger().Info("Ignore saved hardwareInfo, refresh hardware info and generate a new fingerprint")
	} else if !hasFingerprint(savedHwInfo) {
		log.GetLogger().Info("There is no saved hardwareInfo, generate a new fingerprint")
	} else {
		log.GetLogger().Info("Current hardwareInfo is different from saved hardwareInfo, generate a new fingerprint")
	}
	newFingerprint := generateUUID()
	// save hardwareInfo
	if err = SaveHardwareInfo(newFingerprint, hardwareHash); err != nil {
		log.GetLogger().Errorf("Save hardware hash failed: %s", err)
	}
	return newFingerprint, err
}

func generateUUID() string {
	id := uuid.New()
	return fingerprintVersionPrefix + strings.Replace(id.String(), "-", "", -1)
}

// isSimilarHardwareHash
func isSimilarHardwareHash(savedHwHash map[string]string, currentHwHash map[string]string, threshold int) bool {

	var totalCount, successCount int
	isSimilar := true

	// similarity check is disabled when threshold is set to -1
	if threshold == -1 {
		log.GetLogger().Info("Similarity check is disabled, skipping hardware comparison")
		return true
	}

	// check input
	if len(savedHwHash) == 0 {
		log.GetLogger().Errorf("Invalid saved hardware hash: %v", savedHwHash)
		return false
	} else if len(currentHwHash) == 0 {
		log.GetLogger().Errorf("Invalid current hardware hash: %v", savedHwHash)
		return false
	}

	// check whether hardwareId (uuid/machineid) has changed
	// this usually happens during provisioning
	if currentHwHash[hardwareID] != savedHwHash[hardwareID] {
		log.GetLogger().Infof("MachindId/UUID in current hardwareInfo[%s] is different from saved hardwareInfo[%s], "+
			"it's a new instance.", currentHwHash[hardwareID], savedHwHash[hardwareID])
		isSimilar = false

	} else {

		mismatchedKeyMessages := make([]string, 0, len(currentHwHash))
		const unmatchedValueFormat = "The '%s' value (%s) has changed from the registered machine configuration value (%s)."
		const matchedValueFormat = "The '%s' value matches the registered machine configuration value."

		// check whether ipaddress is the same - if the machine key and the IP address have not changed, it's the same instance.
		if currentHwHash[ipaddreddInfo] == savedHwHash[ipaddreddInfo] {
			log.GetLogger().Infof(matchedValueFormat, "IP Address")
		} else {
			message := fmt.Sprintf(unmatchedValueFormat, "IP Address", currentHwHash[ipaddreddInfo], savedHwHash[ipaddreddInfo])
			mismatchedKeyMessages = append(mismatchedKeyMessages, message)
			// identify number of successful matches
			for key, currValue := range currentHwHash {
				if prevValue, ok := savedHwHash[key]; ok && currValue == prevValue {
					log.GetLogger().Infof(matchedValueFormat, key)
					successCount++
				} else {
					message := fmt.Sprintf(unmatchedValueFormat, key, currValue, prevValue)
					mismatchedKeyMessages = append(mismatchedKeyMessages, message)
				}
			}
			// check if the changed match exceeds the minimum match percent
			totalCount = len(currentHwHash)
			similarity := float32(successCount) / float32(totalCount) * 100
			if similarity < float32(threshold) {
				log.GetLogger().Errorf("The similarity[%.2f%%] between current hardware and saved is less than the "+
					"threshold[%.2f%%], it's a new instance.", similarity, float32(threshold))
				for _, message := range mismatchedKeyMessages {
					log.GetLogger().Warn(message)
				}
				isSimilar = false
			}
		}
	}

	return isSimilar
}

func commandOutputHash(command string, params ...string) (encodedValue string, value string, err error) {
	var contentBytes []byte
	if contentBytes, err = exec.Command(command, params...).Output(); err == nil {
		value = string(contentBytes) // without encoding
		sum := md5.Sum(contentBytes)
		encodedValue = base64.StdEncoding.EncodeToString(sum[:])
	}
	return
}

func hostnameInfo() (value string, err error) {
	return os.Hostname()
}

func primaryIpInfo() (value string, err error) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "", err
	}
	for _, address := range addrs {
		// check the address type and if it is not a loopback then return it
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String(), nil
			}
		}
	}
	return "", err
}

func macAddrInfo() (value string, err error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}

	for _, i := range ifaces {
		if i.HardwareAddr.String() != "" {
			return i.HardwareAddr.String(), nil
		}
	}

	return "", nil
}

func isValidHardwareHash(hardwareHash map[string]string) bool {
	for _, value := range hardwareHash {
		if !utf8.ValidString(value) {
			return false
		}
	}

	return true
}

func hasFingerprint(info *HardwareInfo) bool {
	return info != nil && info.Fingerprint != ""
}
