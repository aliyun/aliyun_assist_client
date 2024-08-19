package update

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"go.uber.org/atomic"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/aliyun/aliyun_assist_client/thirdparty/sirupsen/logrus"

	"github.com/aliyun/aliyun_assist_client/agent/log"
	"github.com/aliyun/aliyun_assist_client/agent/metrics"
	"github.com/aliyun/aliyun_assist_client/agent/taskengine"
	"github.com/aliyun/aliyun_assist_client/agent/util/osutil"
	"github.com/aliyun/aliyun_assist_client/agent/util/process"
	"github.com/aliyun/aliyun_assist_client/common/pathutil"
	"github.com/aliyun/aliyun_assist_client/common/fileutil"
	libupdate "github.com/aliyun/aliyun_assist_client/common/update"
)

const (
	// MaximumCheckUpdateRetries is the maximum retry count for checking update
	MaximumCheckUpdateRetries = 3
)

const (
	NotUpdating              = "NotUpdating"
	UpdateReasonBoot         = "Boot"
	UpdateReasonPeriod       = "Periodic"
	UpdateReasonKickRollback = "KickRollback"
	UpdateReasonKickupgrade  = "KickUpgrade"
)

var (
	ErrPreparationTimeout   = errors.New("Updating preparation phase timeout")
	ErrUpdatorNotFound      = errors.New("Updator is required but not found")
	ErrUpdatorNotExecutable = errors.New("Updator is not executable")
	ErrUpdateIsDisabled     = errors.New("Updating is disabled")

	errFailurePlaceholder = errors.New("Failed to execute update script via updator but no error returned")
	errTimeoutPlaceholder = errors.New("Executing update script via updator timeout")

	isUpdating *atomic.String = atomic.NewString(NotUpdating)
)

func ExecuteUpdateScriptRunner(updateScriptPath string) {
	log.GetLogger().Infof("Starting updator to execute update script %s", updateScriptPath)
	updatorPath := libupdate.GetUpdatorPathByCurrentProcess()

	exitcode, status, err := process.SyncRunDetached(updatorPath, []string{"--local_install", updateScriptPath}, 120)
	failureContext := map[string]interface{}{
		"updatorPath":      updatorPath,
		"updateScriptPath": updateScriptPath,
		"executionStatus":  process.StrStatus(status),
	}
	if err != nil {
		if exitcode != process.ExitPlaceholderFailed {
			failureContext["exitcode"] = exitcode
		}
		if status == process.Timeout {
			log.GetLogger().WithFields(logrus.Fields(failureContext)).WithError(err).Errorln("Executing update script via updator timeout")
			libupdate.ReportExecuteUpdateScriptRunnerTimeout(err, nil, failureContext)
		} else {
			log.GetLogger().WithFields(logrus.Fields(failureContext)).WithError(err).Errorln("Failed to execute update script via updator")
			libupdate.ReportExecuteUpdateScriptRunnerFailed(err, nil, failureContext)
		}
	} else if status != process.Success {
		failureContext["exitcode"] = exitcode
		if status == process.Timeout {
			log.GetLogger().WithFields(logrus.Fields(failureContext)).Errorln(errTimeoutPlaceholder.Error())
			libupdate.ReportExecuteUpdateScriptRunnerFailed(errTimeoutPlaceholder, nil, failureContext)
		} else {
			log.GetLogger().WithFields(logrus.Fields(failureContext)).Errorln(errFailurePlaceholder.Error())
			libupdate.ReportExecuteUpdateScriptRunnerFailed(errFailurePlaceholder, nil, failureContext)
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

	return safeUpdate(startTime, preparationTimeout, maximumDownloadTimeout, UpdateReasonBoot)
}

// SafeUpdate checks update information and running tasks before invoking updator
func SafeUpdate(preparationTimeout time.Duration, maximumDownloadTimeout time.Duration, updateReason string) error {
	// golang's runtime promised time.Time.Sub() method works like a monotonic
	// clock, so it's safe for timeout calculation.
	startTime := time.Now()

	return safeUpdate(startTime, preparationTimeout, maximumDownloadTimeout, updateReason)
}

func safeUpdate(startTime time.Time, preparationTimeout time.Duration, maximumDownloadTimeout time.Duration, updateReason string) error {
	errmsg := ""
	extrainfo := ""
	defer func() {
		if len(errmsg) > 0 {
			metrics.GetUpdateFailedEvent(
				"errmsg", errmsg,
				"extrainfo", extrainfo,
				"updateReason", updateReason,
			).ReportEvent()
		}
	}()

	if isUpdating.CompareAndSwap(NotUpdating, updateReason) {
		defer isUpdating.Store(NotUpdating)
	} else {
		// At this time, isUpdating may have been modified by other coroutines,
		// but the probability is very small, ignore for now.
		currentUpdateReason := isUpdating.Load()
		errmsg = "Another update routine is running"
		extrainfo = fmt.Sprintf("another updateReason is %s", currentUpdateReason)
		return errors.New(errmsg)
	}

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
	updatorPath := libupdate.GetUpdatorPathByCurrentProcess()
	if !fileutil.CheckFileIsExist(updatorPath) {
		wrapErr := fmt.Errorf("%w: %s does not exist", ErrUpdatorNotFound, updatorPath)
		log.GetLogger().WithError(wrapErr).Errorln("Updating has been disabled due to updator not found")
		errmsg = wrapErr.Error()
		extrainfo = fmt.Sprintf("updatorPath=%s", updatorPath)
		return wrapErr
	}
	if !fileutil.CheckFileIsExecutable(updatorPath) {
		wrapErr := fmt.Errorf("%w: %s is not executable", ErrUpdatorNotExecutable, updatorPath)
		log.GetLogger().WithError(wrapErr).Errorln("Updating has been disabled due to updator is not executable")
		errmsg = wrapErr.Error()
		extrainfo = fmt.Sprintf("updatorPath=%s", updatorPath)
		return wrapErr
	}

	// WARNING: Loose timeout limit: only breaks preparation phase after action
	// finished
	if preparationTimedOut(startTime, preparationTimeout) {
		errmsg = fmt.Sprintf("after CheckFileIsExist timeout: %s", ErrPreparationTimeout.Error())
		extrainfo = fmt.Sprintf("preparationTimeout=%s", preparationTimeout.String())
		return ErrPreparationTimeout
	}

	// 1. Check whether update package is avialable
	updateInfo, err := func() (*libupdate.UpdateCheckResp, error) {
		var lastErr error = nil
		for i := 0; i < MaximumCheckUpdateRetries; i++ {
			updateInfo, err := libupdate.FetchUpdateInfo()
			if err != nil {
				lastErr = err
				log.GetLogger().WithError(err).Errorln("Failed to check update")

				// WARNING: Loose timeout limit: only breaks preparation phase
				// after action finished
				if preparationTimedOut(startTime, preparationTimeout) {
					return nil, ErrPreparationTimeout
				}

				if i < MaximumCheckUpdateRetries-1 {
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
		errmsg = fmt.Sprintf("FetchUpdateInfo err: %s", err.Error())
		extrainfo = fmt.Sprintf("preparationTimeout=%s", preparationTimeout.String())
		return err
	}
	if updateInfo.NeedUpdate == 0 {
		return nil
	}

	// WARNING: Loose timeout limit: only breaks preparation phase after action
	// finished
	if preparationTimedOut(startTime, preparationTimeout) {
		errmsg = fmt.Sprintf("after FetchUpdateInfo timeout: %s", ErrPreparationTimeout.Error())
		extrainfo = fmt.Sprintf("preparationTimeout=%s", preparationTimeout.String())
		return ErrPreparationTimeout
	}

	// 2. Extract package version from url of update package
	// TODO: Extract package version from downloaded package itself, thus remove
	// dependency on url of update package
	newVersion, err := libupdate.ExtractVersionStringFromURL(updateInfo.UpdateInfo.URL)
	if err != nil {
		errmsg = fmt.Sprintf("ExtractVersionStringFromURL error: %s", err.Error())
		extrainfo = fmt.Sprintf("packageURL=%s", updateInfo.UpdateInfo.URL)
		log.GetLogger().WithFields(logrus.Fields{
			"updateInfo": updateInfo,
			"packageURL": updateInfo.UpdateInfo.URL,
		}).WithError(err).Errorln("Failed to extract package version from URL")
		return err
	}

	// Check if target version's directory existed locally
	updateScriptPath, err := checkLocalDirectory(newVersion)
	if err != nil {
		log.GetLogger().WithError(err).Errorf("Check local directory failed")
	} else {
		return executeUpdateScript(updateScriptPath)
	}

	// WARNING: Loose timeout limit: only breaks preparation phase after action
	// finished
	if preparationTimedOut(startTime, preparationTimeout) {
		errmsg = fmt.Sprintf("after checkLocalDirectory timeout: %s", ErrPreparationTimeout.Error())
		extrainfo = fmt.Sprintf("preparationTimeout=%s", preparationTimeout.String())
		return ErrPreparationTimeout
	}

	// 3. Download update package into temporary directory
	tempDir, err := pathutil.GetTempPath()
	if err != nil {
		errmsg = fmt.Sprintf("GetTempPath error: %s", err.Error())
		return err
	}
	tempSavePath := filepath.Join(tempDir, fmt.Sprintf("aliyun_assist_%s.zip", updateInfo.UpdateInfo.Md5))

	downloadTimeout := time.Duration(0)
	if preparationTimeout > 0 {
		elapsedTime := time.Since(startTime)
		if elapsedTime >= preparationTimeout {
			errmsg = fmt.Sprintf("after GetTempPath timeout: %s", ErrPreparationTimeout.Error())
			extrainfo = fmt.Sprintf("preparationTimeout=%s", preparationTimeout.String())
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
			errmsg = fmt.Sprintf("downloadTimeout < 0 error: %s", ErrPreparationTimeout.Error())
			extrainfo = fmt.Sprintf("preparationTimeout=%s", preparationTimeout.String())
			return ErrPreparationTimeout
		}
	}
	err = libupdate.DownloadPackage(updateInfo.UpdateInfo.URL, tempSavePath, downloadTimeout)
	if err != nil {
		log.GetLogger().WithFields(logrus.Fields{
			"updateInfo":     updateInfo,
			"targetSavePath": tempSavePath,
		}).WithError(err).Errorln("Failed to download update package")

		errmsg = fmt.Sprintf("DownloadPackage error: %s", err.Error())
		extrainfo = fmt.Sprintf("url=%s&tempsavepath=%s&downloadtimeout=%s", updateInfo.UpdateInfo.URL, tempSavePath, downloadTimeout.String())
		// Try our best to report error encountered during downloading packages,
		// and REMEMBER: timeout situation of such reporting action MUST be
		// considered into preparation time.
		libupdate.ReportDownloadPackageFailed(err, updateInfo, map[string]interface{}{
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
		errmsg = fmt.Sprintf("after DownloadPackage timeout: %s", ErrPreparationTimeout.Error())
		extrainfo = fmt.Sprintf("preparationTimeout=%s", preparationTimeout.String())
		return ErrPreparationTimeout
	}

	// Actions contained in below function may occupy much CPU, so criticalActionRunning
	// flag is set to indicate perfmon module and would be unset automatically
	// when function ends.
	updateScriptPath, err = func() (string, error) {
		_cpuIntensiveActionRunning.Set()
		defer _cpuIntensiveActionRunning.Clear()

		// Clean downloaded update package under situations described below:
		// * MD5 checksum does not match
		// * MD5 checksums matches but extracting fails
		// * MD5 checksums matches and extraction succeeds
		defer func() {
			// NOTE: Removing downloaded update pacakge would always be performed
			// even when preparation times out. This would be dangerous when IO
			// operation is slow and will block task execution. Review is needed.
			if err := libupdate.RemoveUpdatePackage(tempSavePath); err != nil {
				errmsg = fmt.Sprintf("RemoveUpdatePackage error: %s", err.Error())
				extrainfo = fmt.Sprintf("downloadedPackagePath=%s&packageURL=%s", tempSavePath, updateInfo.UpdateInfo.URL)
				log.GetLogger().WithFields(logrus.Fields{
					"updateInfo":            updateInfo,
					"downloadedPackagePath": tempSavePath,
				}).WithError(err).Errorln("Failed to clean downloaded update package")
				return
			}
			log.GetLogger().Infof("Clean downloaded update package %s", tempSavePath)
		}()

		// 4. Check MD5 checksum of downloaded update package
		if err := libupdate.CompareFileMD5(tempSavePath, updateInfo.UpdateInfo.Md5); err != nil {
			log.GetLogger().WithFields(logrus.Fields{
				"updateInfo":            updateInfo,
				"downloadedPackagePath": tempSavePath,
			}).WithError(err).Errorln("Inconsistent checksum of update package")
			errmsg = fmt.Sprintf("CompareFileMD5 error: %s", err.Error())
			extrainfo = fmt.Sprintf("downloadedPackagePath=%s&md5InUpdateInfo=%s", tempSavePath, updateInfo.UpdateInfo.Md5)

			libupdate.ReportCheckMD5Failed(err, updateInfo, map[string]interface{}{
				"downloadedPackagePath": tempSavePath,
			})

			return "", err
		}
		log.GetLogger().Infof("Package checksum matched with %s", updateInfo.UpdateInfo.Md5)

		// WARNING: Loose timeout limit: only breaks preparation phase after
		// action finished
		if preparationTimedOut(startTime, preparationTimeout) {
			errmsg = fmt.Sprintf("after CompareFileMD5 timeout: %s", ErrPreparationTimeout.Error())
			extrainfo = fmt.Sprintf("preparationTimeout=%s", preparationTimeout.String())
			return "", ErrPreparationTimeout
		}

		// 5. Remove old versions, only preserving no more than two versions after installation
		destinationDir := libupdate.GetInstallDir()
		if err := libupdate.RemoveOldVersion(destinationDir); err != nil {
			log.GetLogger().WithFields(logrus.Fields{
				"destinationDir": destinationDir,
			}).WithError(err).Warnln("Failed to clean old versions, but not abort updating process")
		}

		// WARNING: Loose timeout limit: only breaks preparation phase after
		// action finished
		if preparationTimedOut(startTime, preparationTimeout) {
			errmsg = fmt.Sprintf("after RemoveOldVersion timeout: %s", ErrPreparationTimeout.Error())
			extrainfo = fmt.Sprintf("preparationTimeout=%s", preparationTimeout.String())
			return "", ErrPreparationTimeout
		}

		// 6. Extract downloaded update package directly to install directory
		if err := libupdate.ExtractPackage(tempSavePath, destinationDir); err != nil {
			errmsg = fmt.Sprintf("ExtractPackage error: %s", err.Error())
			extrainfo = fmt.Sprintf("destinationDir=%s&downloadedPackagePath=%s", destinationDir, tempSavePath)
			log.GetLogger().WithFields(logrus.Fields{
				"updateInfo":            updateInfo,
				"downloadedPackagePath": tempSavePath,
				"destinationDir":        destinationDir,
			}).WithError(err).Errorln("Failed to extract update package")

			libupdate.ReportExtractPackageFailed(err, updateInfo, map[string]interface{}{
				"downloadedPackagePath": tempSavePath,
				"destinationDir":        destinationDir,
			})

			return "", err
		}
		log.GetLogger().Infof("Package extracted to %s", destinationDir)

		// WARNING: Loose timeout limit: only breaks preparation phase after
		// action finished
		if preparationTimedOut(startTime, preparationTimeout) {
			errmsg = fmt.Sprintf("after ExtractPackage timeout: %s", ErrPreparationTimeout.Error())
			extrainfo = fmt.Sprintf("preparationTimeout=%s", preparationTimeout.String())
			return "", ErrPreparationTimeout
		}

		// 7. Validate agent executable file format and architecture
		agentPath := libupdate.GetAgentPathByVersion(newVersion)
		if err := libupdate.ValidateExecutable(agentPath); err != nil {
			errmsg = fmt.Sprintf("GetAgentPathByVersion error: %s", err.Error())
			extrainfo = fmt.Sprintf("packageURL=%s&executablePath=%s", updateInfo.UpdateInfo.URL, agentPath)
			log.GetLogger().WithFields(logrus.Fields{
				"updateInfo": updateInfo,
				"packageURL": updateInfo.UpdateInfo.URL,
			}).WithError(err).Errorln("Invalid agent executable downloaded from responded URL")

			libupdate.ReportValidateExecutableFailed(err, updateInfo, map[string]interface{}{
				"packageURL":     updateInfo.UpdateInfo.URL,
				"executablePath": agentPath,
			})

			return "", err
		}

		// WARNING: Loose timeout limit: only breaks preparation phase after
		// action finished
		if preparationTimedOut(startTime, preparationTimeout) {
			errmsg = fmt.Sprintf("after ValidateExecutable timeout: %s", ErrPreparationTimeout.Error())
			extrainfo = fmt.Sprintf("preparationTimeout=%s", preparationTimeout.String())
			return "", ErrPreparationTimeout
		}

		// 8. Construct and return path of update script to be executed
		updateScriptPath := libupdate.GetUpdateScriptPathByVersion(newVersion)
		return updateScriptPath, nil
	}()
	if err != nil {
		return err
	}

	// WARNING: Loose timeout limit: only breaks preparation phase after action
	// finished
	if preparationTimedOut(startTime, preparationTimeout) {
		errmsg = fmt.Sprintf("after updateScriptPath timeout: %s", ErrPreparationTimeout.Error())
		extrainfo = fmt.Sprintf("preparationTimeout=%s", preparationTimeout.String())
		return ErrPreparationTimeout
	}

	log.GetLogger().Infof("Update script of new version is %s", updateScriptPath)

	return executeUpdateScript(updateScriptPath)
}

func RollbackWithLocalDir(newVersion string, updateReason string) error {
	return updateWithLocalDir(newVersion, updateReason)
}

func UpgradeWithLocalDir(newVersion string, updateReason string) error {
	// Pre-check is updating disabled
	updatingDisabled, err := isUpdatingDisabled()
	if err != nil {
		log.GetLogger().WithError(err).Errorln("Error encountered when reading updating disabling configuration")
	}
	if updatingDisabled {
		log.GetLogger().Infoln("Updating has been disabled due to configuration")
		return ErrUpdateIsDisabled
	}
	return updateWithLocalDir(newVersion, updateReason)
}

func updateWithLocalDir(newVersion string, updateReason string) error {
	errmsg := ""
	extrainfo := ""
	defer func() {
		if len(errmsg) > 0 {
			metrics.GetUpdateFailedEvent(
				"errmsg", errmsg,
				"extrainfo", extrainfo,
				"updateReason", updateReason,
			).ReportEvent()
		}
	}()

	if isUpdating.CompareAndSwap(NotUpdating, updateReason) {
		defer isUpdating.Store(NotUpdating)
	} else {
		// At this time, isUpdating may have been modified by other coroutines,
		// but the probability is very small and ignored for now.
		currentUpdateReason := isUpdating.Load()
		errmsg = "Another update routine is running"
		extrainfo = fmt.Sprintf("another updateReason is %s", currentUpdateReason)
		return errors.New(errmsg)
	}

	updateScriptPath, err := checkLocalDirectory(newVersion)
	if err != nil {
		log.GetLogger().WithError(err).Errorf("Check local directory failed")
		return err
	}
	return executeUpdateScript(updateScriptPath)
}

func executeUpdateScript(updateScriptPath string) error {
	// Actions contained in below function may occupy much CPU, so
	// criticalActionRunning flag is set to indicate perfmon module and unset
	// automatically when function ends.
	// NOTE: I know function below contains too much code, HOWEVER under manual
	// test it does breaks updating procedure and crash process. Some day would
	// be better solution for such situation.
	_cpuIntensiveActionRunning.Set()
	defer _cpuIntensiveActionRunning.Clear()

	// Wait for existing tasks to finish
	for guardExitLoop := false; !guardExitLoop; {
		// defer keyword works in function scope, so closure function is neccessary
		guardExitLoop = func() bool {
			// Check any running tasks. Sleep 5 seconds and restart loop if exist
			if taskengine.FetchingTaskCounter.Load() > 0 ||
				taskengine.GetTaskFactory().IsAnyNonPeriodicTaskRunning() {
				time.Sleep(time.Duration(5) * time.Second)
				return false
			}

			// No running tasks: set criticalActionRunning indicator to
			// refuse kick_vm later, and acquire lock to prevent concurrent
			// fetching tasks
			_criticalActionRunning.Set()
			defer _criticalActionRunning.Clear()
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
}

func preparationTimedOut(startTime time.Time, preparationTimeout time.Duration) bool {
	return preparationTimeout > 0 && time.Now().Sub(startTime) >= preparationTimeout
}

func checkLocalDirectory(targetVersion string) (updateScriptPath string, err error) {
	logger := log.GetLogger().WithFields(logrus.Fields{
		"phase":         "CheckLocalDirectory",
		"targetVersion": targetVersion,
	})
	installDir := libupdate.GetInstallDir()
	entries, err := os.ReadDir(installDir)
	if err != nil {
		logger.WithFields(logrus.Fields{
			"installDir": installDir,
		}).WithError(err).Errorln("Failed to read install dir")
		return "", err
	}
	var foundDir bool
	for _, entry := range entries {
		if entry.IsDir() && entry.Name() == targetVersion {
			foundDir = true
			break
		}
	}
	if !foundDir {
		logger.WithFields(logrus.Fields{
			"installDir": installDir,
		}).Errorln("Not found target directory")
		err = fmt.Errorf("not found")
		return "", err
	}
	targetDir := filepath.Join(installDir, targetVersion)
	updateScriptPath, err = checkHashFile(logger, targetDir)
	if err != nil {
		return
	}
	if osutil.GetOsType() != osutil.OSWin {
		versionFile := filepath.Join(installDir, "version")
		err = os.WriteFile(versionFile, []byte(targetVersion), 0600)
		if err != nil {
			logger.WithFields(logrus.Fields{
				"versionFile": versionFile,
			}).WithError(err).Error("Write new version to version file failed")
		}
	}
	return
}

func checkHashFile(logger logrus.FieldLogger, versionDir string) (updateScriptPath string, err error) {
	logger = logger.WithField("versionDir", versionDir)
	hashFile := filepath.Join(versionDir, "config", "hash_file")
	hashFileCorrupted := filepath.Join(versionDir, "config", "hash_file_corrupted")
	defer func() {
		if err != nil {
			// this directory is corrupted
			os.Rename(hashFile, hashFileCorrupted)
		}
	}()
	content, err := os.ReadFile(hashFile)
	if err != nil {
		logger.WithField("hashFile", hashFile).WithError(err).Error("Read hash file failed")
		return
	}
	decoded, err := base64.StdEncoding.DecodeString(string(content))
	if err != nil {
		logger.WithField("hashFile", hashFile).WithError(err).Error("Decode hash file failed")
		return
	}
	hashMap := make(map[string]string)
	err = json.Unmarshal(decoded, &hashMap)
	if err != nil {
		logger.WithField("hashFile", hashFile).WithError(err).Error("Unmarshal hash file failed")
		return
	}
	var updateScriptFound bool
	updateScript := libupdate.GetUpdateScript()
	for fileName := range hashMap {
		if err = libupdate.CompareFileMD5(filepath.Join(versionDir, fileName), hashMap[fileName]); err != nil {
			logger.WithField("fileName", fileName).WithError(err).Error("MD5 failed")
			return
		}
		if !updateScriptFound && fileName == updateScript {
			updateScriptFound = true
		}
	}
	if updateScriptFound {
		updateScriptPath = filepath.Join(versionDir, updateScript)
		logger.Info("Check hash file success")
	} else {
		logger.WithField("updateScript", updateScript).Error("Not found")
		err = fmt.Errorf("updateScript not found")
	}
	return
}
