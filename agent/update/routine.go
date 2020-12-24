package update

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/aliyun/aliyun_assist_client/agent/clientreport"
	"github.com/aliyun/aliyun_assist_client/agent/log"
	"github.com/aliyun/aliyun_assist_client/agent/taskengine"
	"github.com/aliyun/aliyun_assist_client/agent/util"
	"github.com/aliyun/aliyun_assist_client/agent/util/process"
)

const (
	// MaximumCheckUpdateRetries is the maximum retry count for checking update
	MaximumCheckUpdateRetries = 3
)

func ExecuteUpdator() {
	log.GetLogger().Info("Starting CheckUpdate......")
	updatorPath := GetUpdatorPathByCurrentProcess()

	if err := process.SyncRunSimpleDetached(updatorPath, strings.Split("--check_update", " "), 120); err != nil {
		log.GetLogger().WithError(err).Errorf("Failed to execute updator %s", updatorPath)
	}
}

func ExecuteUpdateScriptRunner(updateScriptPath string) {
	log.GetLogger().Infof("Starting updator to execute update script %s", updateScriptPath)
	updatorPath := GetUpdatorPathByCurrentProcess()

	if err := process.SyncRunSimpleDetached(updatorPath, []string{"--local_install", updateScriptPath}, 120); err != nil {
		log.GetLogger().WithError(err).Errorf("Failed to execute updator %s", updatorPath)

		_, _ = clientreport.ReportUpdateFailure("ExecuteUpdateScriptRunnerFailed", clientreport.UpdateFailure{
			UpdateInfo: nil,
			FailureContext: map[string]interface{}{
				"updateScriptPath": updateScriptPath,
			},
			ErrorMessage: err.Error(),
		})
	}
}

// SafeUpdate checks update information and running tasks before invoking updator
func SafeUpdate() error {
	// 1. Check whether update package is avialable
	updateInfo, err := func () (*UpdateCheckResp, error) {
		var lastErr error = nil
		for i := 0; i < MaximumCheckUpdateRetries; i++ {
			updateInfo, err := FetchUpdateInfo()
			if err != nil {
				lastErr = err
				log.GetLogger().WithError(err).Errorln("Failed to check update")
				if i < MaximumCheckUpdateRetries - 1 {
					time.Sleep(time.Duration(5) * time.Second)
				}
				continue
			}
			return updateInfo, nil
		}
		return nil, lastErr
	}()
	if err != nil {
		return err
	}
	if updateInfo.NeedUpdate == 0 {
		return nil
	}

	// 2. Download update package into temporary directory
	tempDir, err := util.GetTempPath()
	if err != nil {
		return err
	}
	tempSavePath := filepath.Join(tempDir, fmt.Sprintf("aliyun_assist_%s.zip", updateInfo.UpdateInfo.Md5))
	if err := DownloadPackage(updateInfo.UpdateInfo.URL, tempSavePath); err != nil {
		log.GetLogger().WithFields(logrus.Fields{
			"updateInfo": updateInfo,
			"targetSavePath": tempSavePath,
		}).WithError(err).Errorln("Failed to download update package")

		_, _ = clientreport.ReportUpdateFailure("DownloadPackageFailed", clientreport.UpdateFailure{
			UpdateInfo: updateInfo,
			FailureContext: map[string]interface{}{
				"targetSavePath": tempSavePath,
			},
			ErrorMessage: err.Error(),
		})

		return err
	}
	log.GetLogger().Infof("Package downloaded from %s to %s", updateInfo.UpdateInfo.URL, tempSavePath)

	// Actions contained in below function may occupy much CPU, so criticalActionRunning
	// flag is set to indicate perfmon module and would be unset automatically
	// when function ends.
	err = func () error {
		setCriticalActionRunning()
		defer unsetCriticalActionRunning()

		// Clean downloaded update package under situations described below:
		// * MD5 checksum does not match
		// * MD5 checksums matches but extracting fails
		// * MD5 checksums matches and extraction succeeds
		defer func () {
			if err := RemoveUpdatePackage(tempSavePath); err != nil {
				log.GetLogger().WithFields(logrus.Fields{
					"updateInfo": updateInfo,
					"downloadedPackagePath": tempSavePath,
				}).WithError(err).Errorln("Failed to clean downloaded update package")
				return
			}
			log.GetLogger().Infof("Clean downloaded update package %s", tempSavePath)
		}()

		// 3. Check MD5 checksum of downloaded update package
		if err := CompareFileMD5(tempSavePath, updateInfo.UpdateInfo.Md5); err != nil {
			log.GetLogger().WithFields(logrus.Fields{
				"updateInfo": updateInfo,
				"downloadedPackagePath": tempSavePath,
			}).WithError(err).Errorln("Inconsistent checksum of update package")

			_, _ = clientreport.ReportUpdateFailure("CheckMD5Failed", clientreport.UpdateFailure{
				UpdateInfo: updateInfo,
				FailureContext: map[string]interface{}{
					"downloadedPackagePath": tempSavePath,
				},
				ErrorMessage: err.Error(),
			})

			return err
		}
		log.GetLogger().Infof("Package checksum matched with %s", updateInfo.UpdateInfo.Md5)

		// 4. Remove old versions, only preserving no more than two versions after installation
		destinationDir := GetInstallDir()
		if err := RemoveOldVersion(destinationDir); err != nil {
			log.GetLogger().WithFields(logrus.Fields{
				"destinationDir": destinationDir,
			}).WithError(err).Warnln("Failed to clean old versions, but not abort updating process")
		}

		// 5. Extract downloaded update package directly to install directory
		if err := ExtractPackage(tempSavePath, destinationDir); err != nil {
			log.GetLogger().WithFields(logrus.Fields{
				"updateInfo": updateInfo,
				"downloadedPackagePath": tempSavePath,
				"destinationDir": destinationDir,
			}).WithError(err).Errorln("Failed to extract update package")

			_, _ = clientreport.ReportUpdateFailure("ExtractPackageFailed", clientreport.UpdateFailure{
				UpdateInfo: updateInfo,
				FailureContext: map[string]interface{}{
					"downloadedPackagePath": tempSavePath,
					"destinationDir": destinationDir,
				},
				ErrorMessage: err.Error(),
			})

			return err
		}
		log.GetLogger().Infof("Package extracted to %s", destinationDir)

		return nil
	}()
	if err != nil {
		return err
	}

	// 6. Prepare path of update script to be executed
	// TODO: Extract package version from downloaded package itself, thus remove
	// dependency on url of update package
	newVersion, err := ExtractVersionStringFromURL(updateInfo.UpdateInfo.URL)
	if err != nil {
		log.GetLogger().WithFields(logrus.Fields{
			"updateInfo": updateInfo,
			"packageURL": updateInfo.UpdateInfo.URL,
		}).WithError(err).Errorln("Failed to extract package version from URL")
		return err
	}
	updateScriptPath := GetUpdateScriptPathByVersion(newVersion)
	log.GetLogger().Infof("Update script of new version is %s", updateScriptPath)

	// Actions contained in below function may occupy much CPU, so
	// criticalActionRunning flag is set to indicate perfmon module and unset
	// automatically when function ends.
	// NOTE: I know function below contains too much code, HOWEVER under manual
	// test it does breaks updating procedure and crash process. Some day would
	// be better solution for such situation.
	return func () error {
		setCriticalActionRunning()
		defer unsetCriticalActionRunning()

		// 7. Wait for existing tasks to finish
		for guardExitLoop := false; !guardExitLoop; {
			// defer keyword works in function scope, so closure function is neccessary
			guardExitLoop = func() bool {
				// Check any running tasks. Sleep 5 seconds and restart loop if exist
				if taskengine.GetTaskFactory().IsAnyNonPeriodicTaskRunning() {
					time.Sleep(time.Duration(5) * time.Second)
					return false
				}

				// No running tasks: acquire lock to prevent concurrent fetching tasks
				if !taskengine.FetchingTaskLock.TryLock() {
					time.Sleep(time.Duration(5) * time.Second)
					return false
				}
				defer taskengine.FetchingTaskLock.Unlock()

				// Sleep 5 seconds before double check in case fecthing tasks finished just now
				time.Sleep(time.Duration(5) * time.Second)
				// Double check any running tasks
				if taskengine.GetTaskFactory().IsAnyNonPeriodicTaskRunning() {
					// Above updatingMutexGuard should be auto released when function returns
					return false
				}

				// ENSURE: Mutex lock acquired and no running tasks
				ExecuteUpdateScriptRunner(updateScriptPath)
				// Agent process would be killed and code below would never be executed
				return true
			}()
		}

		return nil
	}()
}
