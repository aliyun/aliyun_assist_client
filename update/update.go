package main

import (
	"fmt"
	"time"

	"github.com/aliyun/aliyun_assist_client/agent/log"
	libupdate "github.com/aliyun/aliyun_assist_client/common/update"
)

func doCheckUpdate() error {
	resp, err := libupdate.FetchUpdateInfo()
	if err != nil {
		log.GetLogger().Error("UpdateCheck error: " + err.Error())
		return err
	}

	if resp.NeedUpdate == 0 {
		log.GetLogger().Info("UpdateCheck: No Need to Update")
		return nil
	}

	packageMD5 := resp.UpdateInfo.Md5
	packageURL := resp.UpdateInfo.URL
	log.GetLogger().Info("CheckUpdate:url=", packageURL, " md5=", packageMD5)
	return doUpdate(packageURL, packageMD5, "")
}

func doUpdate(url string, md5 string, version string) error {
	if version == "" {
		extractedVersion, err := libupdate.ExtractVersionStringFromURL(url)
		if err != nil {
			return err
		}
		version = extractedVersion
	}

	// 1. Download update package
	filename := fmt.Sprintf("aliyun-assist_%s.zip", version)
	// NOTE: A timeout of zero means no timeout, and we do not set timeout here
	// since this function is often manually invoked by users.
	if err := libupdate.DownloadPackage(url, filename, time.Duration(0)); err != nil {
		return err
	}

	// 2. Check MD5 checksum of downloaded update package
	if err := libupdate.CompareFileMD5(filename, md5); err != nil {
		return err
	}

	// 3. Clean old versions
	destPath := libupdate.GetInstallDir()
	if err := libupdate.RemoveOldVersion(destPath); err != nil {
		log.GetLogger().WithError(err).Warnf("Failed to clean old versions in %s, but not abort updating process", destPath)
	}

	// 4. Unpack upload package
	if err := libupdate.ExtractPackage(filename, destPath); err != nil {
		return err
	}

	// 5. Validate agent executable file format and architecture
	agentPath := libupdate.GetAgentPathByVersion(version)
	if err := libupdate.ValidateExecutable(agentPath); err != nil {
		return err
	}

	// 6. Execute update script
	updateScriptPath := libupdate.GetUpdateScriptPathByVersion(version)
	if err := doInstall(updateScriptPath); err != nil {
		return err
	}
	log.GetLogger().Infof("Successfully update from %s", url)

	return nil
}

func doInstall(updateScriptPath string) error {
	log.GetLogger().Infof("Executing update script %s", updateScriptPath)

	err := libupdate.ExecuteUpdateScript(updateScriptPath)
	if err != nil {
		libupdate.ReportExecuteUpdateScriptFailed(err, nil, map[string]interface{}{
			"updateScriptPath": updateScriptPath,
		})

		return err
	}

	log.GetLogger().Infof("Successfully execute update script %s", updateScriptPath)
	return nil
}
