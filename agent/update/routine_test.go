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
	libupdate "github.com/aliyun/aliyun_assist_client/common/update"
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
	guard_1 := monkey.Patch(libupdate.FetchUpdateInfo, func() (*libupdate.UpdateCheckResp, error) {
		if !flag {
			flag = true
			return nil, errors.New("some error")
		} else {
			flag = false
			updateCheckResp := libupdate.UpdateCheckResp {
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
			args: theArgs,
		},
		{
			name: "downloadPackageError",
			wantErr: true,
			args: theArgs,
		},
		{
			name: "checkMd5Error",
			wantErr: true,
			args: theArgs,
		},
		{
			name: "removeOldVersionError",
			wantErr: false,
			args: theArgs,
		},
		{
			name: "extractPackageError",
			wantErr: true,
			args: theArgs,
		},
		{
			name: "extractVersionFromUrlError",
			wantErr: true,
			args: theArgs,
		},
		{
			name: "validateExecuteableError",
			wantErr: true,
			args: theArgs,
		},
	}
	guardDisableUpdate := monkey.Patch(isUpdatingDisabled, func() (bool, error) { return false, nil })
	guardDownloadPackage := monkey.Patch(libupdate.DownloadPackage, func(string, string, time.Duration) error { return nil })
	guardCompareFileMD5 := monkey.Patch(libupdate.CompareFileMD5, func(string, string) error { return nil })
	guardRemoveOldVersion := monkey.Patch(libupdate.RemoveOldVersion, func(string) error { return nil })
	guardExtractPackage := monkey.Patch(libupdate.ExtractPackage, func(string, string) error { return nil })
	guardExtractVersionStringFromURL := monkey.Patch(libupdate.ExtractVersionStringFromURL, func(string) (string, error) { return "1.0.0.2", nil})
	guardValidateExecutable := monkey.Patch(libupdate.ValidateExecutable, func(string) error { return nil })
	defer func() {
		guardDisableUpdate.Unpatch()
		guardDownloadPackage.Unpatch()
		guardCompareFileMD5.Unpatch()
		guardRemoveOldVersion.Unpatch()
		guardExtractPackage.Unpatch()
		guardExtractVersionStringFromURL.Unpatch()
		guardValidateExecutable.Unpatch()
	}()

	updatorPath := libupdate.GetUpdatorPathByCurrentProcess()
	if !util.CheckFileIsExist(updatorPath) {
		util.WriteStringToFile(updatorPath, "some")
	}
	defer func() {
		if util.CheckFileIsExist(updatorPath) {
			os.Remove(updatorPath)
		}
	}()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.name == "disableUpdate" {
				guardDisableUpdate.Unpatch()
				guardDisableUpdate = monkey.Patch(isUpdatingDisabled, func() (bool, error) {
					return true, errors.New("some error")
				})
			}

			
			if tt.name == "updatePathNotExist" {
				if util.CheckFileIsExist(updatorPath) {
					os.Remove(updatorPath)
				}
			}

			tempPath := ""
			if tt.name == "downloadPackageError" {
				guardDownloadPackage.Unpatch()
				guardDownloadPackage = monkey.Patch(libupdate.DownloadPackage, func(string, string, time.Duration) error {
					// tempPath = savePath
					return errors.New("some error")
				})
				defer func() {
					if util.CheckFileIsExist(tempPath) {
						os.Remove(tempPath)
					}
				}()
			}
			

			if tt.name == "checkMd5Error" {
				guardCompareFileMD5.Unpatch()
				guardCompareFileMD5 = monkey.Patch(libupdate.CompareFileMD5, func(string, string) error { return errors.New("some error") })
			}

			if tt.name == "removeOldVersionError" {
				guardRemoveOldVersion.Unpatch()
				guardRemoveOldVersion = monkey.Patch(libupdate.RemoveOldVersion, func(string) error { return errors.New("some error") })

				err := libupdate.ExtractPackage("some", "some")
				fmt.Println("ExtractPackage err: ", err)
			}

			if tt.name == "extractPackageError" {
				guardExtractPackage.Unpatch()
				guardExtractPackage = monkey.Patch(libupdate.ExtractPackage, func(string, string) error { return errors.New("some error") })
			}

			if tt.name == "extractVersionFromUrlError" {
				guardExtractVersionStringFromURL.Unpatch()
				guardExtractVersionStringFromURL = monkey.Patch(libupdate.ExtractVersionStringFromURL, func(string) (string, error) { return "", errors.New("some error") })
			}

			if tt.name == "validateExecuteableError" {
				guardValidateExecutable.Unpatch()
				guardValidateExecutable = monkey.Patch(libupdate.ValidateExecutable, func(string) error { return errors.New("some error") })
			}

			if err := safeUpdate(tt.args.startTime, tt.args.preparationTimeout, tt.args.maximumDownloadTimeout); (err != nil) != tt.wantErr {
				t.Errorf("safeUpdate() error = %v, wantErr %v", err, tt.wantErr)
			}
			// // 恢复
			if tt.name == "disableUpdate" {
				guardDisableUpdate.Unpatch()
				guardDisableUpdate = monkey.Patch(isUpdatingDisabled, func() (bool, error) { return false, nil })
			}

			if tt.name == "updatePathNotExist"  {
				if !util.CheckFileIsExist(updatorPath) {
					util.WriteStringToFile(updatorPath, "some")
				}
			}

			if tt.name == "downloadPackageError" {
				guardDownloadPackage.Unpatch()
				guardDownloadPackage = monkey.Patch(libupdate.DownloadPackage, func(string, string, time.Duration) error { return nil })
			}

			if tt.name == "checkMd5Error" {
				guardCompareFileMD5.Unpatch()
				guardCompareFileMD5 = monkey.Patch(libupdate.CompareFileMD5, func(string, string) error { return nil })
			}

			if tt.name == "removeOldVersionError" {
				guardRemoveOldVersion.Unpatch()
				guardRemoveOldVersion = monkey.Patch(libupdate.RemoveOldVersion, func(string) error { return nil })
			}

			if tt.name == "extractPackageError" {
				guardExtractPackage.Unpatch()
				guardExtractPackage = monkey.Patch(libupdate.ExtractPackage, func(string, string) error { return nil })
			}

			if tt.name == "extractVersionFromUrlError" {
				guardExtractVersionStringFromURL.Unpatch()
				guardExtractVersionStringFromURL = monkey.Patch(libupdate.ExtractVersionStringFromURL, func(string) (string, error) { return "1.0.0.2", nil})
			}

			if tt.name == "validateExecuteableError" {
				guardValidateExecutable.Unpatch()
				guardValidateExecutable = monkey.Patch(libupdate.ValidateExecutable, func(string) error { return nil })
			}
		})
	}
}
