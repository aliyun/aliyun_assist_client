//go:build windows

package instance

import (
	"fmt"
	"os"
	"time"
	"path/filepath"

	"github.com/aliyun/aliyun_assist_client/agent/log"
	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/mgr"
	"golang.org/x/sys/windows/registry"
)

const (
	hardwareID     = "uuid"
	wmiServiceName = "Winmgmt"

	serviceRetryInterval = 15 // Seconds
	serviceRetry         = 5
)

func waitForService(service *mgr.Service) error {
	var err error
	var status svc.Status

	for attempt := 1; attempt <= serviceRetry; attempt++ {
		status, err = service.Query()

		if err == nil && status.State == svc.Running {
			return nil
		}
		time.Sleep(serviceRetryInterval * time.Second)
	}

	return fmt.Errorf("failed to wait for WMI to get into Running status")
}

var wmicCommand = filepath.Join(os.Getenv("WINDIR"), "System32", "wbem", "wmic.exe")

var CurrentHwHash = func() (map[string]string, error) {
	hardwareHash := make(map[string]string)

	// Wait for WMI Service
	winManager, err := mgr.Connect()
	if err != nil {
		log.GetLogger().Error("Failed to connect to WMI: ", err)
		return hardwareHash, err
	}

	// Open WMI Service
	var wmiService *mgr.Service
	wmiService, err = winManager.OpenService(wmiServiceName)
	if err != nil {
		log.GetLogger().Error("Failed to open wmi service: ", err)
		return hardwareHash, err
	}

	// Wait for WMI Service to start
	if err = waitForService(wmiService); err != nil {
		log.GetLogger().Error("WMI Service cannot be query for hardware hash.")
		return hardwareHash, err
	}

	log.GetLogger().Info("WMI Service is ready to be queried....")


	hardwareHash[hardwareID], _ = csproductUuid()
	hardwareHash["processor-hash"], _ = processorInfoHash()
	hardwareHash["memory-hash"], _ = memoryInfoHash()
	hardwareHash["bios-hash"], _ = biosInfoHash()
	hardwareHash["system-hash"], _ = systemInfoHash()
	hardwareHash["hostname-info"], _ = hostnameInfo()
	hardwareHash[ipaddreddInfo], _ = primaryIpInfo()
	hardwareHash["macaddr-info"], _ = macAddrInfo()
	hardwareHash["disk-info"], _ = diskInfoHash()

	return hardwareHash, nil
}

// MachineID is the same as the machineID() of common/machineid
func MachineID() (string, error) {
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\Cryptography`, registry.QUERY_VALUE|registry.WOW64_64KEY)
	if err != nil {
		return "", err
	}
	defer k.Close()

	s, _, err := k.GetStringValue("MachineGuid")
	if err != nil {
		return "", err
	}
	return s, nil
}

func csproductUuid() (encodedValue string, err error) {
	encodedValue, _, err = commandOutputHash(wmicCommand, "csproduct", "get", "UUID")
	return
}

func processorInfoHash() (value string, err error) {
	value, _, err = commandOutputHash(wmicCommand, "cpu", "list", "brief")
	return
}

func memoryInfoHash() (value string, err error) {
	value, _, err = commandOutputHash(wmicCommand, "memorychip", "list", "brief")
	return
}

func biosInfoHash() (value string, err error) {
	value, _, err = commandOutputHash(wmicCommand, "bios", "list", "brief")
	return
}

func systemInfoHash() (value string, err error) {
	value, _, err = commandOutputHash(wmicCommand, "computersystem", "list", "brief")
	return
}

func diskInfoHash() (value string, err error) {
	value, _, err = commandOutputHash(wmicCommand, "diskdrive", "list", "brief")
	return
}
