package update

import (
	"errors"
	"io"
	"net/url"
	"os"
	"os/exec"

	"github.com/aliyun/aliyun_assist_client/agent/clientreport"
	"github.com/aliyun/aliyun_assist_client/agent/util/errnoutil"
)

func ReportDownloadPackageFailed(err error, updateInfo *UpdateCheckResp, failureContext map[string]interface{}) {
	failureType := "DownloadPackageFailed"
	if errors.Is(err, os.ErrPermission) {
		failureType += ":AccessDenied"
	} else if urlErr, ok := err.(*url.Error); ok && urlErr.Timeout() {
		failureType += ":NetworkTimeout"
	} else if errors.Is(err, os.ErrDeadlineExceeded) {
		failureType += ":NetworkTimeout"
	} else if errnoutil.IsNoEnoughSpaceError(err) {
		failureType += ":NoEnoughSpace"
	} else if errors.Is(err, io.ErrUnexpectedEOF) {
		failureType += ":UnexpectedEOF"
	}

	clientreport.ReportUpdateFailure(failureType, clientreport.UpdateFailure{
		UpdateInfo: updateInfo,
		FailureContext: failureContext,
		ErrorMessage: err.Error(),
	})
}

func ReportCheckMD5Failed(err error, updateInfo *UpdateCheckResp, failureContext map[string]interface{}) {
	clientreport.ReportUpdateFailure("CheckMD5Failed", clientreport.UpdateFailure{
		UpdateInfo: updateInfo,
		FailureContext: failureContext,
		ErrorMessage: err.Error(),
	})
}

func ReportExtractPackageFailed(err error, updateInfo *UpdateCheckResp, failureContext map[string]interface{}) {
	clientreport.ReportUpdateFailure("ExtractPackageFailed", clientreport.UpdateFailure{
		UpdateInfo: updateInfo,
		FailureContext: failureContext,
		ErrorMessage: err.Error(),
	})
}

func ReportValidateExecutableFailed(err error, updateInfo *UpdateCheckResp, failureContext map[string]interface{}) {
	clientreport.ReportUpdateFailure("ValidateExecutableFailed", clientreport.UpdateFailure{
		UpdateInfo: updateInfo,
		FailureContext: failureContext,
		ErrorMessage: err.Error(),
	})
}

func ReportExecuteUpdateScriptRunnerTimeout(err error, updateInfo *UpdateCheckResp, failureContext map[string]interface{}) {
	reportExecutingFailure("ExecuteUpdateScriptRunnerTimeout", err, updateInfo, failureContext)
}

func ReportExecuteUpdateScriptRunnerFailed(err error, updateInfo *UpdateCheckResp, failureContext map[string]interface{}) {
	reportExecutingFailure("ExecuteUpdateScriptRunnerFailed", err, updateInfo, failureContext)
}

func ReportExecuteUpdateScriptFailed(err error, updateInfo *UpdateCheckResp, failureContext map[string]interface{}) {
	reportExecutingFailure("ExecuteUpdateScriptFailed", err, updateInfo, failureContext)
}

func reportExecutingFailure(failureType string, err error, updateInfo *UpdateCheckResp, failureContext map[string]interface{}) {
	if errors.Is(err, os.ErrNotExist) {
		failureType += ":FileNotExist"
	} else if exitError := (*exec.ExitError)(nil); errors.As(err, &exitError) {
		failureType += categorizeExitCode(exitError.ExitCode())
		if _, ok := failureContext["exitcode"]; !ok {
			failureContext["exitcode"] = exitError.ExitCode()
		}
	} else if err.Error() == "signal: killed" {
		// TODO: Replace error message string matching with error type or
		// attribute assertion
		failureType += ":Killed"
	}

	clientreport.ReportUpdateFailure(failureType, clientreport.UpdateFailure{
		UpdateInfo: updateInfo,
		FailureContext: failureContext,
		ErrorMessage: err.Error(),
	})
}
