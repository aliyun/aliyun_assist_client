package clientreport

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"

	"github.com/aliyun/aliyun_assist_client/agent/util"
)

func TestReportUpdateFailure(t *testing.T) {
	httpmock.Activate()
	util.NilRequest.Set()
	defer util.NilRequest.Clear()
	defer httpmock.DeactivateAndReset()

	const mockRegion = "cn-test100"
	util.MockMetaServer(mockRegion)

	mockResponseBytes, err := json.Marshal(map[string]interface{}{
		"code": 200,
		"errCode": "success",
		"instanceId": "i-test100",
	})
	if err != nil {
		panic(err)
	}

	var requestBody []byte
	httpmock.RegisterResponder("POST",
		fmt.Sprintf("https://%s.axt.aliyun.com/luban/api/v1/exception/client_report", mockRegion),
		func(h *http.Request) (*http.Response, error) {
			readRequestBody, err := ioutil.ReadAll(h.Body)
			if err != nil {
				return nil, err
			}
			requestBody = readRequestBody

			return httpmock.NewBytesResponse(200, mockResponseBytes), nil
		})

	response, err := ReportUpdateFailure("UnitTest", UpdateFailure{
		UpdateInfo: nil,
		FailureContext: map[string]interface{}{"unittest": true},
		ErrorMessage: "UnitTest",
	})
	assert.NoError(t, err, "ReportUpdateFailure should not return error")
	assert.Exactly(t, string(mockResponseBytes), response, "Response should match")

	var sendedReport ClientReport
	assert.NoError(t, json.Unmarshal(requestBody, &sendedReport))
	assert.Exactly(t, sendedReport.ReportType, "AgentUpdateFailure:UnitTest")
	var sendedFailure UpdateFailure
	assert.NoError(t, json.Unmarshal([]byte(sendedReport.Info), &sendedFailure))
	assert.Nil(t, sendedFailure.UpdateInfo)
	assert.Exactly(t, true, sendedFailure.FailureContext["unittest"])
	assert.Exactly(t, sendedFailure.ErrorMessage, "UnitTest")
}

