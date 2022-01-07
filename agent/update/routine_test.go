package update

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"

	"bou.ke/monkey"
	"github.com/aliyun/aliyun_assist_client/agent/util"
	"github.com/jarcoal/httpmock"
)

func Test_safeUpdate(t *testing.T) {
	httpmock.Activate()
	util.NilRequest.Set()
	defer httpmock.DeactivateAndReset()
	defer util.NilRequest.Clear()
	mockRegin := "mock-reginid"
	guard := monkey.Patch(util.GetRegionId, func() string {return mockRegin })
	defer guard.Unpatch()
	httpmock.RegisterResponder("POST",
		fmt.Sprintf("https://%s.axt.aliyun.com/luban/api/metrics", mockRegin),
		func(h *http.Request) (*http.Response, error) {
			return httpmock.NewStringResponse(200, "success"), nil
		})
	flag := false
	guard_1 := monkey.Patch(FetchUpdateInfo, func() (*UpdateCheckResp, error) {
		if !flag {
			flag = true
			return nil, errors.New("some error")
		} else {
			flag = false
			updateCheckResp := UpdateCheckResp {
				Flag: 1,
				InstanceID: "instanceID",
				NeedUpdate: 1,
				NextInterval: 300,
			}
			updateCheckResp.UpdateInfo.FileName = "filename"
			updateCheckResp.UpdateInfo.Md5 = "md5"
			updateCheckResp.UpdateInfo.URL = "url"
			return &updateCheckResp, nil
		}
	})
	defer guard_1.Unpatch()

	type args struct {
		startTime              time.Time
		preparationTimeout     time.Duration
		maximumDownloadTimeout time.Duration
	}
	theArgs := args{
		startTime: time.Now(),
		preparationTimeout: time.Duration(20) * time.Minute,
		maximumDownloadTimeout: time.Duration(10) * time.Minute,
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "disableUpdate",
			wantErr: false,
			args: theArgs,
		},
		{
			name: "updatePathNotExist",
			wantErr: true,
		},
		{
			name: "downloadPackageError",
			wantErr: true,
		},
		{
			name: "checkMd5Error",
			wantErr: true,
		},
		{
			name: "removeOldVersionError",
			wantErr: false,
		},
		{
			name: "extractPackageError",
			wantErr: true,
		},
		{
			name: "extractVersionFromUrlError",
			wantErr: true,
		},
		{
			name: "validateExecuteableError",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.name == "disableUpdate" {
				guard_1 := monkey.Patch(isUpdatingDisabled, func() (bool, error) {
					return true, errors.New("some error")
				})
				defer guard_1.Unpatch()
			} else {
				guard_1 := monkey.Patch(isUpdatingDisabled, func() (bool, error) {
					return false, nil
				})
				defer guard_1.Unpatch()
			}

			updatorPath := GetUpdatorPathByCurrentProcess()
			if tt.name == "updatePathNotExist" {
				if util.CheckFileIsExist(updatorPath) {
					os.Remove(updatorPath)
				}
			} else {
				if !util.CheckFileIsExist(updatorPath) {
					util.WriteStringToFile(updatorPath, "some")
				}
				defer func() {
					if util.CheckFileIsExist(updatorPath) {
						os.Remove(updatorPath)
					}
				}()
			}

			tempPath := ""
			if tt.name == "downloadPackageError" {
				guard := monkey.Patch(DownloadPackage, func(string, string, time.Duration) error {
					// tempPath = savePath
					return errors.New("some error")
				})
				defer guard.Unpatch()
			} else {	
				guard := monkey.Patch(DownloadPackage, func(string, string, time.Duration) error {
					// tempPath = savePath
					return nil
				})
				defer guard.Unpatch()
			}
			defer func() {
				if util.CheckFileIsExist(tempPath) {
					os.Remove(tempPath)
				}
			}()

			if tt.name == "checkMd5Error" {
				guard := monkey.Patch(CompareFileMD5, func(string, string) error { return errors.New("some error") })
				defer guard.Unpatch()
			} else {
				guard := monkey.Patch(CompareFileMD5, func(string, string) error { return nil })
				defer guard.Unpatch()
			}

			if tt.name == "removeOldVersionError" {
				guard := monkey.Patch(RemoveOldVersion, func(string) error { return errors.New("some error") })
				defer guard.Unpatch()
			} else {
				guard := monkey.Patch(RemoveOldVersion, func(string) error { return nil })
				defer guard.Unpatch()
			}

			if tt.name == "extractPackageError" {
				guard := monkey.Patch(ExtractPackage, func(string, string) error { return errors.New("some error") })
				defer guard.Unpatch()
			} else {
				guard := monkey.Patch(ExtractPackage, func(string, string) error { return nil })
				defer guard.Unpatch()
			}

			if tt.name == "extractVersionFromUrlError" {
				guard := monkey.Patch(ExtractVersionStringFromURL, func(string) (string, error) { return "", errors.New("some error") })
				defer guard.Unpatch()
			} else {
				guard := monkey.Patch(ExtractVersionStringFromURL, func(string) (string, error) { return "1.0.0.2", nil})
				defer guard.Unpatch()
			}

			if tt.name == "validateExecuteableError" {
				guard := monkey.Patch(ValidateExecutable, func(string) error { return errors.New("some error") })
				defer guard.Unpatch()
			} else {
				guard := monkey.Patch(ValidateExecutable, func(string) error { return nil })
				defer guard.Unpatch()
			}

			if err := safeUpdate(tt.args.startTime, tt.args.preparationTimeout, tt.args.maximumDownloadTimeout); (err != nil) != tt.wantErr {
				t.Errorf("safeUpdate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
