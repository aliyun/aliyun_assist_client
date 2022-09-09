package taskengine

import (
	"errors"
	"testing"

	"bou.ke/monkey"
	"github.com/aliyun/aliyun_assist_client/agent/util"
	"github.com/aliyun/aliyun_assist_client/agent/taskengine/models"
	"github.com/jarcoal/httpmock"
)

func TestSendFileFinished(t *testing.T) {
	mockMetrics()
	defer util.NilRequest.Clear()
	defer httpmock.DeactivateAndReset()
	guard := monkey.Patch(util.HttpPost, func(string, string, string) (string, error) {
		return "", errors.New("some error")
	})
	defer guard.Unpatch()
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
	guard := monkey.Patch(util.HttpPost, func(string, string, string) (string, error) {
		return "", errors.New("some error")
	})
	defer guard.Unpatch()
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
