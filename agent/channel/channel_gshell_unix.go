//go:build !windows
// +build !windows

package channel

import (
	"fmt"
	"io/fs"
	"io/ioutil"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/aliyun/aliyun_assist_client/agent/log"
	"github.com/aliyun/aliyun_assist_client/common/fileutil"
)

const (
	classVirtioPortPath = "/sys/class/virtio-ports/"
)

var (
	virtportRegexp, _ = regexp.Compile(`^vport.*`)
)

func getGshellPath() (gshellPath string, err error) {
	gshellPath = "/dev/virtio-ports/org.qemu.guest_agent.0"
	if !fileutil.CheckFileIsExist(gshellPath) {
		log.GetLogger().Warnf("gshellPath `%s` not exist, try to find gshellPath from %s", gshellPath, classVirtioPortPath)
		gshellPath = getVportDevPath()
		if gshellPath == "" {
			log.GetLogger().Error("gshell path not found")
			err = fs.ErrNotExist
		}
	}
	return
}

func getVportDevPath() (gshellPath string) {
	rd, err := ioutil.ReadDir(classVirtioPortPath)
	if err != nil {
		log.GetLogger().Errorf("read dir `%s` error: %v", classVirtioPortPath, err)
		return
	}
	for _, fi := range rd {
		if virtportRegexp.MatchString(fi.Name()) {
			nameFile := filepath.Join(classVirtioPortPath, fi.Name(), "name")
			content, err := ioutil.ReadFile(nameFile)
			if err != nil {
				continue
			}
			if strings.TrimSpace(string(content)) == "org.qemu.guest_agent.0" {
				gshellPath = fmt.Sprintf("/dev/%s", fi.Name())
				return
			}
		}
	}
	return
}