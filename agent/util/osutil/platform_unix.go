// +build darwin freebsd linux netbsd openbsd

package osutil

import (
	"os/exec"
	"strings"
	"runtime"

	ini "gopkg.in/ini.v1"

	"github.com/aliyun/aliyun_assist_client/agent/log"
)


const (
	osReleaseFile          = "/etc/os-release"
	systemReleaseFile      = "/etc/system-release"
	centosReleaseFile      = "/etc/centos-release"
	redhatReleaseFile      = "/etc/redhat-release"
	unameCommand           = "/usr/bin/uname"
	lsbReleaseCommand      = "lsb_release"
	fetchingDetailsMessage = "fetching platform details from %v"
	errorOccurredMessage   = "There was an error running %v, err: %v"
)

var osArch string

type osRelease struct {
	NAME       string
	VERSION_ID string
}

func getPlatformName() (value string, err error) {
	value, _, err = getPlatformDetails()
	return
}

func getPlatformType() (value string, err error) {
	return "linux", nil
}

func getPlatformVersion() (value string, err error) {
	_, value, err = getPlatformDetails()
	return
}

func getPlatformDetails() (name string, version string, err error) {
	contents := ""
	var contentsBytes []byte
	name = notAvailableMessage
	version = notAvailableMessage

	if Exists(centosReleaseFile) {
		// CentOS has incomplete information in the osReleaseFile
		// and there fore needs to be before osReleaseFile exist check
		log.GetLogger().Debugf(fetchingDetailsMessage, centosReleaseFile)
		contents, err = ReadFile(centosReleaseFile)
		log.GetLogger().Debugf(commandOutputMessage, contents)

		if err != nil {
			log.GetLogger().Debugf(errorOccurredMessage, centosReleaseFile, err)
			return
		}

		if strings.Contains(contents, "CentOS") || strings.Contains(contents, "Aliyun") || strings.Contains(contents, "Alibaba"){
			data := strings.Split(contents, "release")
			name = strings.TrimSpace(data[0])
			if len(data) >= 2 {
				versionData := strings.Split(data[1], "(")
				version = strings.TrimSpace(versionData[0])
			}
		}
	} else if Exists(osReleaseFile) {

		log.GetLogger().Debugf(fetchingDetailsMessage, osReleaseFile)
		contents := new(osRelease)
		err = ini.MapTo(contents, osReleaseFile)
		log.GetLogger().Debugf(commandOutputMessage, contents)
		if err != nil {
			log.GetLogger().Debugf(errorOccurredMessage, osReleaseFile, err)
			return
		}

		name = contents.NAME
		version = contents.VERSION_ID

	} else if Exists(systemReleaseFile) {
		// We want to fall back to legacy behaviour in case some older versions of
		// linux distributions do not have the or-release file
		log.GetLogger().Debugf(fetchingDetailsMessage, systemReleaseFile)

		contents, err = ReadFile(systemReleaseFile)
		log.GetLogger().Debugf(commandOutputMessage, contents)

		if err != nil {
			log.GetLogger().Debugf(errorOccurredMessage, systemReleaseFile, err)
			return
		}
		if strings.Contains(contents, "Aliyun") {
			data := strings.Split(contents, "release")
			name = strings.TrimSpace(data[0])
			if len(data) >= 2 {
				version = strings.TrimSpace(data[1])
			}
		} else if strings.Contains(contents, "Alibaba") {
			data := strings.Split(contents, "release")
			name = strings.TrimSpace(data[0])
			if len(data) >= 2 {
				version = strings.TrimSpace(data[1])
			}
		} else if strings.Contains(contents, "Red Hat") {
			data := strings.Split(contents, "release")
			name = strings.TrimSpace(data[0])
			if len(data) >= 2 {
				version = strings.TrimSpace(data[1])
			}
		} else if strings.Contains(contents, "CentOS") {
			data := strings.Split(contents, "release")
			name = strings.TrimSpace(data[0])
			if len(data) >= 2 {
				version = strings.TrimSpace(data[1])
			}
		} else if strings.Contains(contents, "SLES") {
			data := strings.Split(contents, "release")
			name = strings.TrimSpace(data[0])
			if len(data) >= 2 {
				version = strings.TrimSpace(data[1])
			}
		} else if strings.Contains(contents, "Raspbian") {
			data := strings.Split(contents, "release")
			name = strings.TrimSpace(data[0])
			if len(data) >= 2 {
				version = strings.TrimSpace(data[1])
			}
		} else if strings.Contains(contents, "Oracle") {
			data := strings.Split(contents, "release")
			name = strings.TrimSpace(data[0])
			if len(data) >= 2 {
				version = strings.TrimSpace(data[1])
			}
		}
	} else if Exists(redhatReleaseFile) {
		log.GetLogger().Debugf(fetchingDetailsMessage, redhatReleaseFile)

		contents, err = ReadFile(redhatReleaseFile)
		log.GetLogger().Debugf(commandOutputMessage, contents)

		if err != nil {
			log.GetLogger().Debugf(errorOccurredMessage, redhatReleaseFile, err)
			return
		}
		if strings.Contains(contents, "Red Hat") {
			data := strings.Split(contents, "release")
			name = strings.TrimSpace(data[0])
			if len(data) >= 2 {
				versionData := strings.Split(data[1], "(")
				version = strings.TrimSpace(versionData[0])
			}
		}
	} else if runtime.GOOS == "freebsd" {
		log.GetLogger().Debugf(fetchingDetailsMessage, unameCommand)

		if contentsBytes, err = exec.Command(unameCommand, "-sr").Output(); err != nil {
			log.GetLogger().Debugf(errorOccurredMessage, unameCommand, err)
			return
		}
		log.GetLogger().Debugf(commandOutputMessage, contentsBytes)

		data := strings.Split(string(contentsBytes), " ")
		name = strings.TrimSpace(data[0])
		if len(data) >= 2 {
			version = strings.TrimSpace(data[1])
		}
	} else if runtime.GOOS == "darwin" {
		version = "1.0.0"
		name = "MacOS"
	} else {
		log.GetLogger().Debugf(fetchingDetailsMessage, lsbReleaseCommand)

		// platform name
		if contentsBytes, err = exec.Command(lsbReleaseCommand, "-i").Output(); err != nil {
			log.GetLogger().Debugf(errorOccurredMessage, lsbReleaseCommand, err)
			return
		}
		name = strings.TrimSpace(string(contentsBytes))
		log.GetLogger().Debugf(commandOutputMessage, name)
		name = strings.TrimSpace(string(contentsBytes))
		name = strings.TrimLeft(name, "Distributor ID:")
		name = strings.TrimSpace(name)
		log.GetLogger().Debugf("platform name %v", name)

		// platform version
		if contentsBytes, err = exec.Command(lsbReleaseCommand, "-r").Output(); err != nil {
			log.GetLogger().Debugf(errorOccurredMessage, lsbReleaseCommand, err)
			return
		}
		version = strings.TrimSpace(string(contentsBytes))
		log.GetLogger().Debugf(commandOutputMessage, version)
		version = strings.TrimLeft(version, "Release:")
		version = strings.TrimSpace(version)
		log.GetLogger().Debugf("platform version %v", version)
	}
	return
}

func getArch() (formatArch string) {
	if osArch != "" {
		return osArch
	}
	defer func() {
		osArch = formatArch
	}()
	formatArch = ARCH_UNKNOWN
	arch, err := GetUnameMachine()
	if err != nil {
		log.GetLogger().Errorln("Get Arch: GetUnameMachine err: ", err.Error())
		return ARCH_UNKNOWN
	}
	arch = strings.TrimSpace(arch)
	arch = strings.ToLower(arch)

	if strings.Contains(arch, "aarch") || strings.Contains(arch, "arm") { // arm: aarch arm
		formatArch = ARCH_ARM
	} else if strings.Contains(arch, "386") || strings.Contains(arch, "686") { // x86: i386 i686
		formatArch = ARCH_32
	} else if arch == "x86_64" { // x64: x86_64
		formatArch = ARCH_64
	} else {
		log.GetLogger().Errorln("Get Arch: unknown arch: ", arch)
		formatArch = ARCH_UNKNOWN
	}
	return
}