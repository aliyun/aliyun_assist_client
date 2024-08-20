//go:build !windows

package instance

import (
	"os"
	"strings"
)

const (
	dmidecodeCommand = "/usr/sbin/dmidecode"
	hardwareID       = "machine-id"

	dbusPath = "/var/lib/dbus/machine-id"
	dbusPathEtc = "/etc/machine-id"
	uuidPath = "/sys/class/dmi/id/product_uuid"
)

var CurrentHwHash = func() (map[string]string, error) {
	hardwareHash := make(map[string]string)
	hardwareHash[hardwareID], _ = MachineID()
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
	id, err := os.ReadFile(dbusPath)
	if err != nil {
		// try fallback path
		id, err = os.ReadFile(dbusPathEtc)
	}
	if err != nil {
		id, err = os.ReadFile(uuidPath)
	}
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(id)), nil
}

func processorInfoHash() (value string, err error) {
	value, _, err = commandOutputHash(dmidecodeCommand, "-t", "processor")
	return
}

func memoryInfoHash() (value string, err error) {
	value, _, err = commandOutputHash(dmidecodeCommand, "-t", "memory")
	return
}

func biosInfoHash() (value string, err error) {
	value, _, err = commandOutputHash(dmidecodeCommand, "-t", "bios")
	return
}

func systemInfoHash() (value string, err error) {
	value, _, err = commandOutputHash(dmidecodeCommand, "-t", "system")
	return
}

func diskInfoHash() (value string, err error) {
	value, _, err = commandOutputHash("ls", "-l", "/dev/disk/by-uuid")
	return
}
