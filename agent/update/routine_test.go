package update

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"reflect"
	"testing"
	"time"

	gomonkey "github.com/agiledragon/gomonkey/v2"
	"github.com/aliyun/aliyun_assist_client/agent/clientreport"
	"github.com/aliyun/aliyun_assist_client/agent/metrics"
	"github.com/aliyun/aliyun_assist_client/agent/util"
	"github.com/aliyun/aliyun_assist_client/common/fileutil"
	libupdate "github.com/aliyun/aliyun_assist_client/common/update"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
)

func Test_safeUpdate(t *testing.T) {
	httpmock.Activate()
	util.NilRequest.Set()
	defer httpmock.DeactivateAndReset()
	defer util.NilRequest.Clear()
	mockRegin := "mock-reginid"
	guard := gomonkey.ApplyFunc(util.GetRegionId, func() string { return mockRegin })
	defer guard.Reset()
	httpmock.RegisterResponder("POST",
		fmt.Sprintf("https://%s.axt.aliyun.com/luban/api/metrics", mockRegin),
		func(h *http.Request) (*http.Response, error) {
			return httpmock.NewStringResponse(200, "success"), nil
		})
	flag := false
	guard_1 := gomonkey.ApplyFunc(libupdate.FetchUpdateInfo, func() (*libupdate.UpdateCheckResp, error) {
		if !flag {
			flag = true
			return nil, errors.New("some error")
		} else {
			flag = false
			updateCheckResp := libupdate.UpdateCheckResp{
				Flag:         1,
				InstanceID:   "instanceID",
				NeedUpdate:   1,
				NextInterval: 300,
			}
			updateCheckResp.UpdateInfo.FileName = "filename"
			updateCheckResp.UpdateInfo.Md5 = "md5"
			updateCheckResp.UpdateInfo.URL = "url"
			return &updateCheckResp, nil
		}
	})
	defer guard_1.Reset()

	var m *metrics.MetricsEvent
	guard_2 := gomonkey.ApplyMethod(reflect.TypeOf(m), "ReportEvent", func(me *metrics.MetricsEvent) {
		keywords := make(map[string]string)
		json.Unmarshal([]byte(me.KeyWords), &keywords)
		fmt.Println("------------------")
		for k, v := range keywords {
			fmt.Printf("%s: %s\n", k, v)
		}
	})
	defer guard_2.Reset()

	guard_3 := gomonkey.ApplyFunc(clientreport.ReportUpdateFailure, func(failureType string, failure clientreport.UpdateFailure) (string, error) {
		fmt.Printf("%s: %v\n", failureType, failure)
		return "", nil
	})
	defer guard_3.Reset()

	type args struct {
		startTime              time.Time
		preparationTimeout     time.Duration
		maximumDownloadTimeout time.Duration
	}
	theArgs := args{
		startTime:              time.Now(),
		preparationTimeout:     time.Duration(20) * time.Minute,
		maximumDownloadTimeout: time.Duration(10) * time.Minute,
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name:    "disableUpdate",
			wantErr: false,
			args:    theArgs,
		},
		{
			name:    "updatePathNotExist",
			wantErr: true,
			args:    theArgs,
		},
		{
			name:    "downloadPackageError",
			wantErr: true,
			args:    theArgs,
		},
		{
			name:    "checkMd5Error",
			wantErr: true,
			args:    theArgs,
		},
		{
			name:    "removeOldVersionError",
			wantErr: false,
			args:    theArgs,
		},
		{
			name:    "extractPackageError",
			wantErr: true,
			args:    theArgs,
		},
		{
			name:    "extractVersionFromUrlError",
			wantErr: true,
			args:    theArgs,
		},
		{
			name:    "validateExecuteableError",
			wantErr: true,
			args:    theArgs,
		},
	}
	guardDisableUpdate := gomonkey.ApplyFunc(isUpdatingDisabled, func() (bool, error) { return false, nil })
	guardDownloadPackage := gomonkey.ApplyFunc(libupdate.DownloadPackage, func(string, string, time.Duration) error { return nil })
	guardCompareFileMD5 := gomonkey.ApplyFunc(libupdate.CompareFileMD5, func(string, string) error { return nil })
	guardRemoveOldVersion := gomonkey.ApplyFunc(libupdate.RemoveOldVersion, func(string) error { return nil })
	guardExtractPackage := gomonkey.ApplyFunc(libupdate.ExtractPackage, func(string, string) error { return nil })
	guardExtractVersionStringFromURL := gomonkey.ApplyFunc(libupdate.ExtractVersionStringFromURL, func(string) (string, error) { return "1.0.0.2", nil })
	guardValidateExecutable := gomonkey.ApplyFunc(libupdate.ValidateExecutable, func(string) error { return nil })
	defer func() {
		guardDisableUpdate.Reset()
		guardDownloadPackage.Reset()
		guardCompareFileMD5.Reset()
		guardRemoveOldVersion.Reset()
		guardExtractPackage.Reset()
		guardExtractVersionStringFromURL.Reset()
		guardValidateExecutable.Reset()
	}()

	updatorPath := libupdate.GetUpdatorPathByCurrentProcess()
	if !fileutil.CheckFileIsExist(updatorPath) {
		err := fileutil.WriteStringToFile(updatorPath, "some")
		assert.Equal(t, nil, err)
	}
	os.Chmod(updatorPath, 0766)
	defer func() {
		if fileutil.CheckFileIsExist(updatorPath) {
			os.Remove(updatorPath)
		}
	}()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.name == "disableUpdate" {
				guardDisableUpdate.Reset()
				guardDisableUpdate = gomonkey.ApplyFunc(isUpdatingDisabled, func() (bool, error) {
					return true, errors.New("some error")
				})
			}

			if tt.name == "updatePathNotExist" {
				if fileutil.CheckFileIsExist(updatorPath) {
					os.Remove(updatorPath)
				}
			}

			tempPath := ""
			if tt.name == "downloadPackageError" {
				guardDownloadPackage.Reset()
				guardDownloadPackage = gomonkey.ApplyFunc(libupdate.DownloadPackage, func(string, string, time.Duration) error {
					// tempPath = savePath
					return errors.New("some error")
				})
				defer func() {
					if fileutil.CheckFileIsExist(tempPath) {
						os.Remove(tempPath)
					}
				}()
			}

			if tt.name == "checkMd5Error" {
				guardCompareFileMD5.Reset()
				guardCompareFileMD5 = gomonkey.ApplyFunc(libupdate.CompareFileMD5, func(string, string) error { return errors.New("some error") })
			}

			if tt.name == "removeOldVersionError" {
				guardRemoveOldVersion.Reset()
				guardRemoveOldVersion = gomonkey.ApplyFunc(libupdate.RemoveOldVersion, func(string) error { return errors.New("some error") })

				err := libupdate.ExtractPackage("some", "some")
				fmt.Println("ExtractPackage err: ", err)
			}

			if tt.name == "extractPackageError" {
				guardExtractPackage.Reset()
				guardExtractPackage = gomonkey.ApplyFunc(libupdate.ExtractPackage, func(string, string) error { return errors.New("some error") })
			}

			if tt.name == "extractVersionFromUrlError" {
				guardExtractVersionStringFromURL.Reset()
				guardExtractVersionStringFromURL = gomonkey.ApplyFunc(libupdate.ExtractVersionStringFromURL, func(string) (string, error) { return "", errors.New("some error") })
			}

			if tt.name == "validateExecuteableError" {
				guardValidateExecutable.Reset()
				guardValidateExecutable = gomonkey.ApplyFunc(libupdate.ValidateExecutable, func(string) error { return errors.New("some error") })
			}

			if err := safeUpdate(tt.args.startTime, tt.args.preparationTimeout, tt.args.maximumDownloadTimeout, UpdateReasonPeriod); (err != nil) != tt.wantErr {
				t.Errorf("safeUpdate() error = %v, wantErr %v", err, tt.wantErr)
			}
			// // 恢复
			if tt.name == "disableUpdate" {
				guardDisableUpdate.Reset()
				guardDisableUpdate = gomonkey.ApplyFunc(isUpdatingDisabled, func() (bool, error) { return false, nil })
			}

			if tt.name == "updatePathNotExist" {
				if !fileutil.CheckFileIsExist(updatorPath) {
					fileutil.WriteStringToFile(updatorPath, "some")
					os.Chmod(updatorPath, 0766)
				}
			}

			if tt.name == "downloadPackageError" {
				guardDownloadPackage.Reset()
				guardDownloadPackage = gomonkey.ApplyFunc(libupdate.DownloadPackage, func(string, string, time.Duration) error { return nil })
			}

			if tt.name == "checkMd5Error" {
				guardCompareFileMD5.Reset()
				guardCompareFileMD5 = gomonkey.ApplyFunc(libupdate.CompareFileMD5, func(string, string) error { return nil })
			}

			if tt.name == "removeOldVersionError" {
				guardRemoveOldVersion.Reset()
				guardRemoveOldVersion = gomonkey.ApplyFunc(libupdate.RemoveOldVersion, func(string) error { return nil })
			}

			if tt.name == "extractPackageError" {
				guardExtractPackage.Reset()
				guardExtractPackage = gomonkey.ApplyFunc(libupdate.ExtractPackage, func(string, string) error { return nil })
			}

			if tt.name == "extractVersionFromUrlError" {
				guardExtractVersionStringFromURL.Reset()
				guardExtractVersionStringFromURL = gomonkey.ApplyFunc(libupdate.ExtractVersionStringFromURL, func(string) (string, error) { return "1.0.0.2", nil })
			}

			if tt.name == "validateExecuteableError" {
				guardValidateExecutable.Reset()
				guardValidateExecutable = gomonkey.ApplyFunc(libupdate.ValidateExecutable, func(string) error { return nil })
			}
		})
	}
}
