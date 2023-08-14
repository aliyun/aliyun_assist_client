package osutil

import (
	"errors"
	"io/ioutil"
	"net"
	"os"
	"runtime"
	"strings"

	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/host"
)

const (
	OSWin   = "windows"
	OSLinux = "linux"
	OSFreebsd = "freebsd"
)

// GetUptimeOfMs returns system uptime in millisecond precision, which would be
// simply 0 when error occurs internally.
func GetUptimeOfMs() uint64 {
	// Documentation of Uptime() function does not describe its precision, but we
	// find out from detail implementation that it is truncated to seconds.
	uptimeSeconds, _ := host.Uptime()
	return uptimeSeconds * 1000
}

// GetOsType returns simple os identifier
func GetOsType() string {
	if runtime.GOOS == OSWin {
		return OSWin
	} else if runtime.GOOS == OSLinux {
		return OSLinux
	} else if runtime.GOOS == OSFreebsd {
		return OSFreebsd
	} else {
		return "unknown"
	}
}

func GetOsArch() string {
	return getArch()
}

// GetVirtualType returns kvm or xen based on underlying virtualization type,
// otherwise unknown. TODO: retrieve cpu infos only at initialization time.
func GetVirtualType() string {
	const ResultOnFailure = "unknown"
	// cpu.Info() on linux will return 1 item per physical thread according to
	// documentation, so we just pick the first item for vendor string
	cpuInfoList, err := cpu.Info()
	if err != nil {
		return ResultOnFailure
	}
	if len(cpuInfoList) == 0 {
		return ResultOnFailure
	}

	cpuInfo := cpuInfoList[0]
	if strings.Contains(cpuInfo.VendorID, "KVM") {
		return "kvm"
	} else if strings.Contains(cpuInfo.VendorID, "Xen") {
		return "xen"
	} else {
		return ResultOnFailure
	}
}

func ExternalIP() (net.IP, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 {
			continue // interface down
		}
		if iface.Flags&net.FlagLoopback != 0 {
			continue // loopback interface
		}
		addrs, err := iface.Addrs()
		if err != nil {
			return nil, err
		}
		for _, addr := range addrs {
			ip := getIpFromAddr(addr)
			if ip == nil {
				continue
			}
			return ip, nil
		}
	}
	return nil, errors.New("connected to the network?")
}

func getIpFromAddr(addr net.Addr) net.IP {
	var ip net.IP
	switch v := addr.(type) {
	case *net.IPNet:
		ip = v.IP
	case *net.IPAddr:
		ip = v.IP
	}
	if ip == nil || ip.IsLoopback() {
		return nil
	}
	ip = ip.To4()
	if ip == nil {
		return nil
	}

	return ip
}

func Exists(path string) bool {
	_, err := os.Stat(path) //os.Stat获取文件信息
	if err != nil {
		if os.IsExist(err) {
			return true
		}
		return false
	}
	return true
}

func ReadFile(filename string) (content string, err error) {
	f, err := os.Open(filename)
	if err != nil {
		return
	}
	b, err := ioutil.ReadAll(f)
	if err != nil {
		return
	}
	content = string(b)
	return
}

func WriteFile(filename string, content string) (err error) {
	index := strings.LastIndex(filename, "/")
	if index < 0 {
		index = strings.LastIndex(filename, "\\")
	}
	if index < 0 {
		return errors.New(`error: Can't find "/" or "\".`)
	}
	path := filename[:index]
	if !Exists(path) {
		err = os.MkdirAll(path, os.ModePerm)
		if err != nil {
			return
		}
	}
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	f.WriteString(content)
	f.Close()
	return
}

func DeleteFile(name string) (err error) {
	err = os.Remove(name)
	if err != nil {
		return
	}
	return
}

func GetCpuCores() (cores int, err error) {
	cpuInfoList, err := cpu.Info()
	if err != nil {
		return 0, err
	}
	for _, cpuInfo := range cpuInfoList {
		cores = cores + int(cpuInfo.Cores)
	}
	return
}
