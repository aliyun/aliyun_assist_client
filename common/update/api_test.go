package update

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	gomonkey "github.com/agiledragon/gomonkey/v2"
	"github.com/aliyun/aliyun_assist_client/agent/util"
	"github.com/jarcoal/httpmock"
	"github.com/aliyun/aliyun_assist_client/common/requester"
	"github.com/aliyun/aliyun_assist_client/thirdparty/sirupsen/logrus"
	"github.com/aliyun/aliyun_assist_client/internal/testutil"
)

func TestFetchUpdateInfo(t *testing.T) {
	guard_transport := gomonkey.ApplyFunc(requester.GetHTTPTransport, func(logrus.FieldLogger) *http.Transport {
		transport, _ := http.DefaultTransport.(*http.Transport)
		return transport
	})
	defer guard_transport.Reset()

	httpmock.Activate()
	util.NilRequest.Set()
	defer httpmock.DeactivateAndReset()
	defer util.NilRequest.Clear()
	mockReginid := "test-regin"
	testutil.MockMetaServer(mockReginid)
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
