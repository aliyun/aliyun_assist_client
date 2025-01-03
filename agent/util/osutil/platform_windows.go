//go:build windows
// +build windows

package osutil

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/aliyun/aliyun_assist_client/agent/log"
	"github.com/aliyun/aliyun_assist_client/common/executil"
	"golang.org/x/sys/windows/registry"
)

const caption = "Caption"
const version = "Version"

func getPlatformName() (value string, err error) {
	return getPlatformDetails(caption)
}

func getPlatformType() (value string, err error) {
	return "windows", nil
}

func getPlatformVersion() (value string, err error) {
	value, err = getPlatformDetails(version)
	if err != nil {
		return
	}
	ubr := getUpdateBuildRevision()
	if len(ubr) > 0 {
		value = value + "." + ubr
	}
	return
}

func getPlatformDetails(property string) (value string, err error) {
	log.GetLogger().Debug(gettingPlatformDetailsMessage)
	value = notAvailableMessage

	cmdName := "wmic"
	cmdArgs := []string{"OS", "get", property, "/format:list"}
	var cmdOut []byte
	if cmdOut, err = executil.Command(cmdName, cmdArgs...).Output(); err != nil {
		log.GetLogger().Debugf("There was an error running %v %v, err:%v", cmdName, cmdArgs, err)
		return
	}

	// Stringnize cmd output and trim spaces
	value = strings.TrimSpace(string(cmdOut))

	// Match whitespaces between property and = sign and remove whitespaces
	rp := regexp.MustCompile(fmt.Sprintf("%v(\\s*)%v", property, "="))
	value = rp.ReplaceAllString(value, "")

	// Trim spaces again
	value = strings.TrimSpace(value)

	log.GetLogger().Debugf(commandOutputMessage, value)
	return
}

func getArch() (formatArch string) {
	// 云助手的windows版架构只有amd64的
	return ARCH_64
}

func getUpdateBuildRevision() string {
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\Windows NT\CurrentVersion`, registry.QUERY_VALUE|registry.WOW64_64KEY)
	if err != nil {
		return ""
	}
	defer k.Close()

	ubr, _, err := k.GetIntegerValue("UBR")
	if err != nil {
		return ""
	}
	return fmt.Sprint(ubr)
}
