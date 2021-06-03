package update

import (
	"errors"
	"fmt"
	"net/url"
	"path/filepath"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/aliyun/aliyun_assist_client/agent/log"
	"github.com/aliyun/aliyun_assist_client/agent/taskengine"
	"github.com/aliyun/aliyun_assist_client/agent/util"
	"github.com/aliyun/aliyun_assist_client/agent/util/process"
)

const (
	// MaximumCheckUpdateRetries is the maximum retry count for checking update
	MaximumCheckUpdateRetries = 3
)

var (
	ErrPreparationTimeout = errors.New("Updating preparation phase timeout")
	ErrUpdatorNotFound = errors.New("Updator is required but not found")

	errFailurePlaceholder = errors.New("Failed to execute update script via updator but no error returned")
	errTimeoutPlaceholder = errors.New("Executing update script via updator timeout")
)

func ExecuteUpdateScriptRunner(updateScriptPath string) {
	log.GetLogger().Infof("Starting updator to execute update script %s", updateScriptPath)
	updatorPath := GetUpdatorPathByCurrentProcess()

	exitcode, status, err := process.SyncRunDetached(updatorPath, []string{"--local_install", updateScriptPath}, 120)
	failureContext := map[string]interface{}{
		"updatorPath": updatorPath,
		"updateScriptPath": updateScriptPath,
		"executionStatus": process.StrStatus(status),
	}
	if err != nil {
		if exitcode != process.ExitPlaceholderFailed {
			failureContext["exitcode"] = exitcode
		}
		if status == process.Timeout {
			log.GetLogger().WithFields(logrus.Fields(failureContext)).WithError(err).Errorln("Executing update script via updator timeout")
			ReportExecuteUpdateScriptRunnerTimeout(err, nil, failureContext)
		} else {
			log.GetLogger().WithFields(logrus.Fields(failureContext)).WithError(err).Errorln("Failed to execute update script via updator")
			ReportExecuteUpdateScriptRunnerFailed(err, nil, failureContext)
		}
	} else if status != process.Success {
		failureContext["exitcode"] = exitcode
		if status == process.Timeout {
			log.GetLogger().WithFields(logrus.Fields(failureContext)).Errorln(errTimeoutPlaceholder.Error())
			ReportExecuteUpdateScriptRunnerFailed(errTimeoutPlaceholder, nil, failureContext)
		} else {
			log.GetLogger().WithFields(logrus.Fields(failureContext)).Errorln(errFailurePlaceholder.Error())
			ReportExecuteUpdateScriptRunnerFailed(errFailurePlaceholder, nil, failureContext)
		}
	}
}

func SafeBootstrapUpdate(preparationTimeout time.Duration, maximumDownloadTimeout time.Duration) error {
	// golang's runtime promised time.Time.Sub() method works like a monotonic
	// clock, so it's safe for timeout calculation.
	startTime := time.Now()

	// 0. Pre-check
	boostrapUpdatingDisabled, err := isBootstrapUpdatingDisabled()
	if err != nil {
		log.GetLogger().WithError(err).Errorln("Error encountered when reading bootstrap updating disabling configuration")
	}
	if boostrapUpdatingDisabled {
		log.GetLogger().Infoln("Bootstrap updating has been disabled due to configuration")
		return nil
	}

	// WARNING: Loose timeout limit: only breaks preparation phase after action
	// finished
	if preparationTimedOut(startTime, preparationTimeout) {
		return ErrPreparationTimeout
	}

	return safeUpdate(startTime, preparationTimeout, maximumDownloadTimeout)
}

// SafeUpdate checks update information and running tasks before invoking updator
func SafeUpdate(preparationTimeout time.Duration, maximumDownloadTimeout time.Duration) error {
	// golang's runtime promised time.Time.Sub() method works like a monotonic
	// clock, so it's safe for timeout calculation.
	startTime := time.Now()

	return safeUpdate(startTime, preparationTimeout, maximumDownloadTimeout)
}

func safeUpdate(startTime time.Time, preparationTimeout time.Duration, maximumDownloadTimeout time.Duration) error {
	// 0. Pre-check
	updatingDisabled, err := isUpdatingDisabled()
	if err != nil {
		log.GetLogger().WithError(err).Errorln("Error encountered when reading updating disabling configuration")
	}
	if updatingDisabled {
		log.GetLogger().Infoln("Updating has been disabled due to configuration")
		return nil
	}
	// Check updator existence for possible disabling, compatibile with 1.* version
	updatorPath := GetUpdatorPathByCurrentProcess()
	if !util.CheckFileIsExist(updatorPath) {
		wrapErr := fmt.Errorf("%w: %s does not exist", ErrUpdatorNotFound, updatorPath)
		log.GetLogger().WithError(wrapErr).Errorln("Updating has been disabled due to updator not found")
		return wrapErr
	}

	// WARNING: Loose timeout limit: only breaks preparation phase after action
	// finished
	if preparationTimedOut(startTime, preparationTimeout) {
		return ErrPreparationTimeout
	}

	// 1. Check whether update package is avialable
	updateInfo, err := func () (*UpdateCheckResp, error) {
		var lastErr error = nil
		for i := 0; i < MaximumCheckUpdateRetries; i++ {
			updateInfo, err := FetchUpdateInfo()
			if err != nil {
				lastErr = err
				log.GetLogger().WithError(err).Errorln("Failed to check update")

				// WARNING: Loose timeout limit: only breaks preparation phase
				// after action finished
				if preparationTimedOut(startTime, preparationTimeout) {
					return nil, ErrPreparationTimeout
				}

				if i < MaximumCheckUpdateRetries - 1 {
					time.Sleep(time.Duration(5) * time.Second)
				}

				// WARNING: Loose timeout limit: only breaks preparation phase
				// after action finished
				if preparationTimedOut(startTime, preparationTimeout) {
					return nil, ErrPreparationTimeout
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

	// WARNING: Loose timeout limit: only breaks preparation phase after action
	// finished
	if preparationTimedOut(startTime, preparationTimeout) {
		return ErrPreparationTimeout
	}

	// 2. Download update package into temporary directory
	tempDir, err := util.GetTempPath()
	if err != nil {
		return err
	}
	tempSavePath := filepath.Join(tempDir, fmt.Sprintf("aliyun_assist_%s.zip", updateInfo.UpdateInfo.Md5))

	downloadTimeout := time.Duration(0)
	if preparationTimeout > 0 {
		elapsedTime := time.Now().Sub(startTime)
		if elapsedTime >= preparationTimeout {
			return ErrPreparationTimeout
		}

		downloadTimeout = preparationTimeout - elapsedTime
		if maximumDownloadTimeout < downloadTimeout {
			downloadTimeout = maximumDownloadTimeout
		}

		// Error encountered during downloading packages would be tried to
		// report even when network timeout, thus some time in the remaining
		// MUST be reserved for it.
		// TODO: Update timeout reservation to be consistent with timeout
		// settings in HTTP utilites
		downloadTimeout -= time.Duration(5) * time.Second

		// Re-check downloadTimeout value in case of negative value after subtraction
		if downloadTimeout < 0 {
			return ErrPreparationTimeout
		}
	}
	err = DownloadPackage(updateInfo.UpdateInfo.URL, tempSavePath, downloadTimeout)
	if err != nil {
		log.GetLogger().WithFields(logrus.Fields{
			"updateInfo": updateInfo,
			"targetSavePath": tempSavePath,
		}).WithError(err).Errorln("Failed to download update package")

		// Try our best to report error encountered during downloading packages,
		// and REMEMBER: timeout situation of such reporting action MUST be
		// considered into preparation time.
		ReportDownloadPackageFailed(err, updateInfo, map[string]interface{}{
			"targetSavePath": tempSavePath,
		})

		if timeoutURLErr, ok := err.(*url.Error); ok && timeoutURLErr.Timeout() {
			return ErrPreparationTimeout
		} else {
			return err
		}
	}
	log.GetLogger().Infof("Package downloaded from %s to %s", updateInfo.UpdateInfo.URL, tempSavePath)

	// WARNING: Loose timeout limit: only breaks preparation phase after action
	// finished
	if preparationTimedOut(startTime, preparationTimeout) {
		return ErrPreparationTimeout
	}

	// Actions contained in below function may occupy much CPU, so criticalActionRunning
	// flag is set to indicate perfmon module and would be unset automatically
	// when function ends.
	err = func () error {
		_cpuIntensiveActionRunning.Set()
		defer _cpuIntensiveActionRunning.Clear()

		// Clean downloaded update package under situations described below:
		// * MD5 checksum does not match
		// * MD5 checksums matches but extracting fails
		// * MD5 checksums matches and extraction succeeds
		defer func () {
			// NOTE: Removing downloaded update pacakge would always be performed
			// even when preparation times out. This would be dangerous when IO
			// operation is slow and will block task execution. Review is needed.
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

			ReportCheckMD5Failed(err, updateInfo, map[string]interface{}{
				"downloadedPackagePath": tempSavePath,
			})

			return err
		}
		log.GetLogger().Infof("Package checksum matched with %s", updateInfo.UpdateInfo.Md5)

		// WARNING: Loose timeout limit: only breaks preparation phase after
		// action finished
		if preparationTimedOut(startTime, preparationTimeout) {
			return ErrPreparationTimeout
		}

		// 4. Remove old versions, only preserving no more than two versions after installation
		destinationDir := GetInstallDir()
		if err := RemoveOldVersion(destinationDir); err != nil {
			log.GetLogger().WithFields(logrus.Fields{
				"destinationDir": destinationDir,
			}).WithError(err).Warnln("Failed to clean old versions, but not abort updating process")
		}

		// WARNING: Loose timeout limit: only breaks preparation phase after
		// action finished
		if preparationTimedOut(startTime, preparationTimeout) {
			return ErrPreparationTimeout
		}

		// 5. Extract downloaded update package directly to install directory
		if err := ExtractPackage(tempSavePath, destinationDir); err != nil {
			log.GetLogger().WithFields(logrus.Fields{
				"updateInfo": updateInfo,
				"downloadedPackagePath": tempSavePath,
				"destinationDir": destinationDir,
			}).WithError(err).Errorln("Failed to extract update package")

			ReportExtractPackageFailed(err, updateInfo, map[string]interface{}{
				"downloadedPackagePath": tempSavePath,
				"destinationDir": destinationDir,
			})

			return err
		}
		log.GetLogger().Infof("Package extracted to %s", destinationDir)

		// WARNING: Loose timeout limit: only breaks preparation phase after
		// action finished
		if preparationTimedOut(startTime, preparationTimeout) {
			return ErrPreparationTimeout
		}

		return nil
	}()
	if err != nil {
		return err
	}

	// WARNING: Loose timeout limit: only breaks preparation phase after action
	// finished
	if preparationTimedOut(startTime, preparationTimeout) {
		return ErrPreparationTimeout
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
		_cpuIntensiveActionRunning.Set()
		defer _cpuIntensiveActionRunning.Clear()
		_criticalActionRunning.Set()
		defer _criticalActionRunning.Clear()

		// 7. Wait for existing tasks to finish
		for guardExitLoop := false; !guardExitLoop; {
			// defer keyword works in function scope, so closure function is neccessary
			guardExitLoop = func() bool {
				// Check any running tasks. Sleep 5 seconds and restart loop if exist
				if taskengine.FetchingTaskCounter.Load() > 0 ||
					taskengine.GetTaskFactory().IsAnyNonPeriodicTaskRunning() {
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
				if taskengine.FetchingTaskCounter.Load() > 0 ||
					taskengine.GetTaskFactory().IsAnyNonPeriodicTaskRunning() {
					// Above updatingMutexGuard should be auto released when function returns
					return false
				}

				// ENSURE: Mutex lock acquired and no running tasks
				// NOTE: No strict timeout should be set for updating script
				// execution, preventing updating script is killed after stopping
				// service action is issued, which would cause agent of new version
				// cannot be started correctly.
				ExecuteUpdateScriptRunner(updateScriptPath)
				// Agent process would be killed and code below would never be executed
				return true
			}()
		}

		return nil
	}()
}

func preparationTimedOut(startTime time.Time, preparationTimeout time.Duration) bool {
	return preparationTimeout > 0 && time.Now().Sub(startTime) >= preparationTimeout
}
