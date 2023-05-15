// +build windows

package osutil

import (
	"os/exec"
	"regexp"
	"strings"
	"fmt"

	"github.com/aliyun/aliyun_assist_client/agent/log"
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
	return getPlatformDetails(version)
}

func getPlatformDetails(property string, ) (value string, err error) {
	log.GetLogger().Debug(gettingPlatformDetailsMessage)
	value = notAvailableMessage

	cmdName := "wmic"
	cmdArgs := []string{"OS", "get", property, "/format:list"}
	var cmdOut []byte
	if cmdOut, err = exec.Command(cmdName, cmdArgs...).Output(); err != nil {
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