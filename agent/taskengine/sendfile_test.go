package taskengine

import (
	"testing"
	"net/http"

	gomonkey "github.com/agiledragon/gomonkey/v2"
	"github.com/aliyun/aliyun_assist_client/agent/util"
	"github.com/aliyun/aliyun_assist_client/agent/taskengine/models"
	"github.com/jarcoal/httpmock"
)

func TestSendFileFinished(t *testing.T) {
	mockMetrics()
	defer util.NilRequest.Clear()
	defer httpmock.DeactivateAndReset()

	const mockRegion = "cn-test100"
	guard_GetServerHost := gomonkey.ApplyFunc(util.GetServerHost, func() string {
		return mockRegion + ".axt.aliyun.com"
	})
	defer guard_GetServerHost.Reset()
	httpmock.RegisterResponder("POST",
		util.GetFinishOutputService(),
		func(h *http.Request) (*http.Response, error) {
			return httpmock.NewStringResponse(200, "success"), nil
		})

	type args struct {
		sendFile models.SendFileTaskInfo
		status   int
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "status-success",
			args: args{
				sendFile: models.SendFileTaskInfo{
					TaskID: "abc",
				},
				status: ESuccess,
			},
		},
		{
			name: "status-fail",
			args: args{
				sendFile: models.SendFileTaskInfo{
					TaskID: "abc",
				},
				status: EFileCreateFail,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			SendFileFinished(tt.args.sendFile, tt.args.status)
		})
	}
}

func TestSendFileInvalid(t *testing.T) {
	mockMetrics()
	defer util.NilRequest.Clear()
	defer httpmock.DeactivateAndReset()

	const mockRegion = "cn-test100"
	guard_GetServerHost := gomonkey.ApplyFunc(util.GetServerHost, func() string {
		return mockRegion + ".axt.aliyun.com"
	})
	defer guard_GetServerHost.Reset()
	httpmock.RegisterResponder("POST",
		util.GetInvalidTaskService(),
		func(h *http.Request) (*http.Response, error) {
			return httpmock.NewStringResponse(200, "success"), nil
		})

	type args struct {
		sendFile models.SendFileTaskInfo
		status   int
	}
	type TT struct {
		name string
		args args
	}
	tests := []TT{}
	sendFile := models.SendFileTaskInfo{
		Name:      "abc",
		Signature: "signature",
		Mode:      "mode",
		Group:     "group",
		Owner:     "owner",
	}
	statuslist := []int{
		EInvalidFilePath,
		EFileAlreadyExist,
		EEmptyContent,
		EInvalidContent,
		EInvalidSignature,
		EInalidFileMode,
		EInalidGID,
		EInalidUID,
	}
	for _, status := range statuslist {
		tests = append(tests, TT{
			args: args{
				sendFile: sendFile,
				status:   status,
			},
		})
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			SendFileInvalid(tt.args.sendFile, tt.args.status)
		})
	}
}
