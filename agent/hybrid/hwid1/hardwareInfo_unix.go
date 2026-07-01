//go:build !windows

package hwid1

import (
	"github.com/aliyun/aliyun_assist_client/common/machineid"
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
	hardwareHash[hardwareID], _ = machineid.GetMachineID()
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
