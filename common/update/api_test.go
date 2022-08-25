package update

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"bou.ke/monkey"
	"github.com/aliyun/aliyun_assist_client/agent/util"
	"github.com/jarcoal/httpmock"
)

func TestFetchUpdateInfo(t *testing.T) {
	httpmock.Activate()
	util.NilRequest.Set()
	defer httpmock.DeactivateAndReset()
	defer util.NilRequest.Clear()
	mockReginid := "test-regin"
	guard := monkey.Patch(util.GetRegionId, func() string { return mockReginid })
	defer guard.Unpatch()
	httpmock.RegisterResponder("POST", 
		fmt.Sprintf("https://%s.axt.aliyun.com/luban/api/v1/update/update_check", mockReginid),
		func(h *http.Request) (*http.Response, error) {
			updateCheckResp := UpdateCheckReport{
				Os: "os",
				Arch: "arch",
				OsVersion: "osversion",
				AppID: "appid",
				AppVersion: "appversion",
			}
			content, err := json.Marshal(&updateCheckResp)
			return httpmock.NewStringResponse(200, string(content)), err
		})

	tests := []struct {
		name    string
		want    *UpdateCheckResp
		wantErr bool
	}{
		{
			name: "normal",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := FetchUpdateInfo()
			if (err != nil) != tt.wantErr {
				t.Errorf("FetchUpdateInfo() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}
