package update

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/aliyun/aliyun_assist_client/agent/log"
	"github.com/aliyun/aliyun_assist_client/agent/util"
	"github.com/aliyun/aliyun_assist_client/agent/util/langutil"
	"github.com/aliyun/aliyun_assist_client/agent/util/process"
	"github.com/aliyun/aliyun_assist_client/agent/util/versionutil"
	"github.com/aliyun/aliyun_assist_client/agent/version"
	"github.com/aliyun/aliyun_assist_client/common/zipfile"
)

const (
	regexVersionPattern = "^\\d+(\\.\\d+){3}$"
)

func ExtractVersionStringFromURL(url string) (string, error) {
	parts := strings.Split(url, "/")
	if len(parts) < 2 {
		return "", errors.New("invalid url:" + url)
	}
	names := strings.Split(parts[len(parts) - 1], "_")
	if len(names) != 2 {
		return "", errors.New("invalid url:" + url)
	}
	version := names[0]
	return version, nil
}


func DownloadPackage(url string, savePath string, timeout time.Duration) error {
	return util.HttpDownloadWithTimeout(url, savePath, timeout)
}


func CompareFileMD5(filePath string, expectedMD5 string) error {
	computedMD5, err := util.ComputeMd5(filePath)
	if err != nil {
		return err
	}

	if strings.ToLower(computedMD5) != strings.ToLower(expectedMD5) {
		return fmt.Errorf("Inconsistent MD5 checksum for %s: expecting %s, actually %s", filePath, expectedMD5, computedMD5)
	}
	return nil
}


func ExtractPackage(filePath string, destination string) error {
	return zipfile.Unzip(filePath, destination)
}


func RemoveUpdatePackage(tempSavePath string) error {
	return os.Remove(tempSavePath)
}


func ExecuteUpdateScript(updateScriptPath string) error {
	if runtime.GOOS != "windows" {
		err, _, _ := util.ExeCmd("chmod +x " + updateScriptPath)
		if err != nil {
			log.GetLogger().Errorln("Failed to add executable permission for update script:", err)
		}
	}

	// TODO: Refactor code below as utility function
	var cmd *exec.Cmd
	if util.G_IsWindows {
		cmd = exec.Command("cmd", "/c", updateScriptPath)
	} else {
		cmd = exec.Command("sh", "-c", updateScriptPath)
	}
	var combinedBuffer process.SafeBuffer
	cmd.Stdout = &combinedBuffer
	cmd.Stderr = &combinedBuffer
	err := cmd.Run()

	log.GetLogger().Info("Update script executed: ", langutil.LocalToUTF8(combinedBuffer.String()))
	if err != nil {
		log.GetLogger().Info("Update script executed. err: ", err)
		return err
	}

	return nil
}

func RemoveOldVersion(multipleVersionDir string) error {
	log.GetLogger().Infof("Removing old version in %s", multipleVersionDir)
	if multipleVersionDir == "" {
		return errors.New("Install dir is empty")
	}

	dir, err := os.Open(multipleVersionDir)
	if err != nil {
		return err
	}
	defer dir.Close()

	children, err := dir.Readdir(-1)
	if err != nil {
		return err
	}

	regexPattern, err := regexp.Compile(regexVersionPattern)
	if err != nil {
		return err
	}

	for _, child := range children {
		name := child.Name()
		if name == "." || name == ".." {
			continue
		}

		if !child.IsDir() {
			continue
		}

		if !regexPattern.MatchString(name) {
			continue
		}

		if versionutil.CompareVersion(name, version.AssistVersion) < 0 {
			outdatedDir := filepath.Join(multipleVersionDir, name)
			log.GetLogger().Infof("Remove old version: %s", outdatedDir)
			if err := os.RemoveAll(outdatedDir); err != nil {
				log.GetLogger().WithError(err).Errorf("Error encountered when removing old version: %s", outdatedDir)
			}
		}
	}

	return nil
}
