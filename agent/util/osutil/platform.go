package osutil

import (
	"os"
	"regexp"
	"runtime"

	"github.com/aliyun/aliyun_assist_client/agent/log"
)

const (
	gettingPlatformDetailsMessage = "getting platform details"
	notAvailableMessage           = "NotAvailable"
	commandOutputMessage          = "Command output %v"
)

const (
	ARCH_64      = "amd64"
	ARCH_32      = "386"
	ARCH_ARM     = "arm64"
	ARCH_UNKNOWN = "unknown"
)

// PlatformFamily marks a family of similar operating systems

// PlatformFamilyWindows uses Ohai identifier for windows platform family
const PlatformFamilyWindows = "windows"

// PlatformFamilyDarwin uses Ohai identifier for darwin platform family
const PlatformFamilyDarwin = "mac_os_x"

// PlatformFamilyDebian uses Ohai identifier for debian platform family
const PlatformFamilyDebian = "debian"

// PlatformFamilyRhel uses Ohai identifier for rhel platform family
const PlatformFamilyRhel = "rhel"

// PlatformFamilyFedora uses Ohai identifier for fedora platform family
const PlatformFamilyFedora = "fedora"

// PlatformFamilyAlpine uses Ohai identifier for alpine platform family
const PlatformFamilyAlpine = "alpine"

// PlatformFamilySuse uses Ohai identifier for opensuse platform family
const PlatformFamilySuse = "suse"

// PlatformFamilyGentoo uses Ohai identifier for gentoo linux platform family
const PlatformFamilyGentoo = "gentoo"

// PlatformFamilyArch uses Ohai identifier for arch linux platform family
const PlatformFamilyArch = "arch"

// Platform marks a specific operating systems

// PlatformDebian uses Ohai identifier for debian platform
const PlatformDebian = "debian"

// PlatformUbuntu uses Ohai identifier for ubuntu platform
const PlatformUbuntu = "ubuntu"

// PlatformRaspbian uses Ohai identifier for raspbian platform
const PlatformRaspbian = "raspbian"

// PlatformRedhat uses Ohai identifier for redhat platform
const PlatformRedhat = "redhat"

// PlatformOracleLinux uses Ohai identifier for oracle linux platform
const PlatformOracleLinux = "oracle"

// PlatformCentos uses Ohai identifier for centos platform
const PlatformCentos = "centos"

// PlatformFedora uses Ohai identifier for fedora platform
const PlatformFedora = "fedora"

// PlatformAliyun uses Ohai identifier for aliyun platform
const PlatformAliyun = "aliyun"

// PlatformAlpine uses Ohai identifier for alpine platform
const PlatformAlpine = "alpine"

// PlatformSuse uses Ohai identifier for suse platform
const PlatformSuse = "suse"

// PlatformOpensuse uses Ohai identifier for opensuse platform version < 42
const PlatformOpensuse = "opensuse"

// PlatformOpensuseLeap uses Ohai identifier for aliyun platform version >= 42
const PlatformOpensuseLeap = "opensuseleap"

// PlatformGentoo uses Ohai identifier for gentoo platform
const PlatformGentoo = "gentoo"

// PlatformArch uses Ohai identifier for arch platform
const PlatformArch = "arch"

// PlatformWindows uses Ohai identifier for windows platform
const PlatformWindows = "windows"

// PlatformDarwin uses Ohai identifier for darwin platform
const PlatformDarwin = "mac_os_x"

// PlatformName gets the OS specific platform name.
func PlatformName() (name string, err error) {
	platform, err := getPlatformName()
	if err != nil {
		return
	}
	return getNormalizedPlatform(platform)
}

// OriginPlatformName gets the original platform name returned by OS
func OriginPlatformName() (name string, err error) {
	return getPlatformName()
}

// PlatformVersion gets the OS specific platform version.
func PlatformVersion() (version string, err error) {
	return getPlatformVersion()
}

func PlatformType() (value string, err error) {
	return getPlatformType()
}

func PlatformArchitect() (value string, err error) {
	if runtime.GOARCH == "amd64" {
		return "x86_64", nil
	}
	return runtime.GOARCH, nil
}

func getNormalizedPlatform(data string) (string, error) {
	mapping := []struct {
		regex    string
		platform string
	}{
		{`(?i)Red Hat Enterprise Linux`, PlatformRedhat},
		{`(?i)Oracle Linux`, PlatformOracleLinux},
		{`(?i)CentOS( Linux)?`, PlatformCentos},
		{`(?i)Fedora( Linux)?`, PlatformFedora},
		{`(?i)Aliyun Linux`, PlatformAliyun},
		{"Debian", PlatformDebian},
		{"Ubuntu", PlatformUbuntu},
		{"(?i)Windows?", PlatformWindows},
		{"(openSUSE Leap)|(SLES)", PlatformSuse},
	}

	for _, m := range mapping {
		if regexp.MustCompile(m.regex).MatchString(data) {
			return m.platform, nil
		}
	}

	return data, nil
}

func GetHostname() string {
	hostname, err := os.Hostname()
	if err != nil {
		log.GetLogger().WithError(err).Errorln("get hostname failed")
	}
	return hostname
}
