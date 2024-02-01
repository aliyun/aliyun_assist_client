package checkvirt

import (
	"errors"
	"fmt"
	"path"
	"strconv"
	"syscall"
	"unsafe"

	"github.com/aliyun/aliyun_assist_client/agent/log"
	"github.com/aliyun/aliyun_assist_client/agent/metrics"
	"github.com/aliyun/aliyun_assist_client/agent/taskengine/timermanager"
	"golang.org/x/sys/windows"
)

const (
	// DefaultCheckIntervalSeconds is the default interval for report virtio driver version
	DefaultCheckIntervalSeconds = 3600 * 24
)
const (
	IOCTL_STORAGE_QUERY_PROPERTY  = 0x002d1400
	StorageDeviceUniqueIdProperty = 3
	PropertyStandardQuery         = 0
	StorageIdCodeSetAscii         = 2
	StorageIdCodeSetBinary        = 1
	StorageIdTypeEUI64            = 2
)

type STORAGE_DEVICE_UNIQUE_IDENTIFIER struct {
	Version                    uint32
	Size                       uint32
	StorageDeviceIdOffset      uint32
	StorageDeviceOffset        uint32
	DriveLayoutSignatureOffset uint32
	raw                        [512]byte
}

type STORAGE_PROPERTY_QUERY struct {
	PropertyId           uint32
	QueryType            uint32
	AdditionalParameters byte
}

type STORAGE_DEVICE_ID_DESCRIPTOR struct {
	Version             uint32
	Size                uint32
	NumberOfIdentifiers uint32
	Identifiers         [1]STORAGE_IDENTIFIER
}

type STORAGE_IDENTIFIER struct {
	CodeSet        uint32
	Type           uint32
	IdentifierSize uint16
	NextOffset     uint16
	Association    uint32
	Identifier     [1]byte
}

func GetDiskPropertyDUID(path string, diskProperty []byte) error {
	utfPath, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return err
	}
	hFile, err := syscall.CreateFile(utfPath,
		syscall.GENERIC_READ,
		syscall.FILE_SHARE_READ|syscall.FILE_SHARE_WRITE,
		nil,
		syscall.OPEN_EXISTING,
		0,
		0)

	if err != nil {
		return err
	}
	defer syscall.CloseHandle(hFile)
	var diskdiskPropertySize uint32

	query := &STORAGE_PROPERTY_QUERY{
		AdditionalParameters: 0,
		PropertyId:           StorageDeviceUniqueIdProperty,
		QueryType:            PropertyStandardQuery,
	}
	return syscall.DeviceIoControl(hFile,
		IOCTL_STORAGE_QUERY_PROPERTY,
		(*byte)(unsafe.Pointer(query)),
		uint32(unsafe.Sizeof(*query)),
		(*byte)(unsafe.Pointer(&diskProperty[0])),
		uint32(len(diskProperty)),
		&diskdiskPropertySize,
		nil)
}

func CheckVirtIoVersion(Index int) (error, bool) {
	diskDuidInfo := make([]byte, 1024, 1024)
	err := GetDiskPropertyDUID("\\\\.\\PhysicalDrive"+strconv.Itoa(Index), diskDuidInfo)
	if err != nil {
		return err, false
	}

	diskData := (*STORAGE_DEVICE_UNIQUE_IDENTIFIER)(unsafe.Pointer(&diskDuidInfo[0]))
	if diskData.StorageDeviceIdOffset >= 1024 {
		return errors.New("Invalid diskData.StorageDeviceIdOffset"), false
	}
	pIdentifiers := (*STORAGE_DEVICE_ID_DESCRIPTOR)(unsafe.Pointer(uintptr(unsafe.Pointer(diskData)) + uintptr(diskData.StorageDeviceIdOffset)))
	if pIdentifiers.NumberOfIdentifiers < 1 {
		return errors.New("Invalid pIdentifiers.NumberOfIdentifiers"), false
	}
	blockStart := uintptr(unsafe.Pointer(&pIdentifiers.Identifiers[0].Identifier[0])) - uintptr(unsafe.Pointer(&diskDuidInfo[0]))
	idSize := pIdentifiers.Identifiers[0].IdentifierSize
	codeSet := pIdentifiers.Identifiers[0].CodeSet
	Type := pIdentifiers.Identifiers[0].Type
	if blockStart+uintptr(idSize) >= 1024 {
		return errors.New("Invalid idSize"), false
	}

	dst := diskDuidInfo[blockStart : blockStart+uintptr(idSize)]
	//老版本驱动uniqueid生成逻辑
	// IdentificationDescr->CodeSet = VpdCodeSetBinary;
	// IdentificationDescr->IdentifierType = VpdIdentifierTypeEUI64;
	// IdentificationDescr->IdentifierLength = 8;
	// IdentificationDescr->Identifier[0] = '1';
	// IdentificationDescr->Identifier[1] = 'A';
	// IdentificationDescr->Identifier[2] = 'F';
	// IdentificationDescr->Identifier[3] = '4';
	// IdentificationDescr->Identifier[4] = '1';
	// IdentificationDescr->Identifier[5] = '0';
	// IdentificationDescr->Identifier[6] = '0';
	// IdentificationDescr->Identifier[7] = '1';
	if Type == StorageIdTypeEUI64 &&
		codeSet == StorageIdCodeSetBinary &&
		idSize == 8 &&
		string(dst[:]) == "1AF41001" {
		log.GetLogger().Println("find dangerous uniqueid:" + string(dst[:]))
		return nil, true
	}
	return nil, false
}

func doCheck() {
	system32, err := windows.GetSystemDirectory()
	if err != nil {
		system32 = "C:\\Windows\\System32"
	}
	driversDir := path.Join(system32, "drivers")
	driversVersionMap := map[string]WinVersion{
		"viostor.sys":       WinVersion{},
		"netkvm.sys":        WinVersion{},
		"vioser.sys":        WinVersion{},
		"AliNVMe.sys":       WinVersion{},
		"AliWinUtilDrv.sys": WinVersion{},
	}
	for key, _ := range driversVersionMap {
		ver, err := GetFileVersion(path.Join(driversDir, key))
		if err != nil {
			log.GetLogger().Infof("get %s version failed,err=%s", key, err.Error())
		} else {
			driversVersionMap[key] = ver
		}
	}
	viostor := driversVersionMap["viostor.sys"]
	netkvm := driversVersionMap["netkvm.sys"]
	vioser := driversVersionMap["vioser.sys"]
	AliNVMe := driversVersionMap["AliNVMe.sys"]
	AliWinUtilDrv := driversVersionMap["AliWinUtilDrv.sys"]
	vminit, err := GetFileVersion("C:\\ProgramData\\aliyun\\vminit\\vminit.exe")
	if err != nil {
		log.GetLogger().Infof("get vminit.exe version failed,err=%s", err.Error())
	}
	metrics.GetVirtioVersionEvent(
		"viostor", fmt.Sprintf("%d.%d.%d.%d", viostor.Major, viostor.Minor, viostor.Patch, viostor.Build),
		"netkvm", fmt.Sprintf("%d.%d.%d.%d", netkvm.Major, netkvm.Minor, netkvm.Patch, netkvm.Build),
		"vioser", fmt.Sprintf("%d.%d.%d.%d", vioser.Major, vioser.Minor, vioser.Patch, vioser.Build),
		"AliNVMe", fmt.Sprintf("%d.%d.%d.%d", AliNVMe.Major, AliNVMe.Minor, AliNVMe.Patch, AliNVMe.Build),
		"AliWinUtilDrv", fmt.Sprintf("%d.%d.%d.%d", AliWinUtilDrv.Major, AliWinUtilDrv.Minor, AliWinUtilDrv.Patch, AliWinUtilDrv.Build),
		"vminit", fmt.Sprintf("%d.%d.%d.%d", vminit.Major, vminit.Minor, vminit.Patch, vminit.Build),
	).ReportEvent()
}

func StartVirtIoVersionReport() error {
	timerManager := timermanager.GetTimerManager()
	timer, err := timerManager.CreateTimerInSeconds(doCheck, DefaultCheckIntervalSeconds)
	if err != nil {
		return err
	}
	_, err = timer.Run()
	if err != nil {
		return err
	}
	return nil
}
