package statemanager

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/aliyun/aliyun_assist_client/agent/util"
	"github.com/aliyun/aliyun_assist_client/agent/util/timetool"
)

type ApiResponse struct {
	ErrCode string `json:"errCode"`
	ErrMsg  string `json:"errMsg"`
}

type StateConfiguration struct {
	StateConfigurationId string
	TemplateName         string
	TemplateVersion      string
	Parameters           string
	ConfigureMode        string
	ScheduleType         string
	ScheduleExpression   string
	SuccessfulApplyTime  string
	DefinitionUpdateTime string
}

type ListInstanceStateConfigurationsResult struct {
	RequestId           string
	Interval            int
	Changed             bool
	Checkpoint          string
	StateConfigurations []StateConfiguration
}

type ListInstanceStateConfigurationsResp struct {
	ApiResponse
	Result *ListInstanceStateConfigurationsResult `json:"result"`
}

type GetTemplateResult struct {
	RequestId       string
	TemplateName    string
	TemplateVersion string
	Content         string
}

type GetTemplateResponse struct {
	ApiResponse
	Result *GetTemplateResult `json:"result"`
}

// ListInstanceStateConfigurations lists state configurations from server and report agent info
func ListInstanceStateConfigurations(lastCheckpoint, agentName, agentVersion, computerName,
	platformName, platformType, platformVersion, ipAddress, ramRole string) (*ListInstanceStateConfigurationsResp, error) {
	url := util.GetStateConfigService()
	var parameters = make(map[string]interface{})
	if len(lastCheckpoint) > 0 {
		parameters = map[string]interface{}{"lastCheckpoint": lastCheckpoint}
	}
	parameters["agentName"] = agentName
	parameters["agentVersion"] = agentVersion
	parameters["computerName"] = computerName
	parameters["platformName"] = platformName
	parameters["platformType"] = platformType
	parameters["platformVersion"] = platformVersion
	parameters["ipAddress"] = ipAddress
	parameters["ramRole"] = ramRole
	resp := &ListInstanceStateConfigurationsResp{}
	err := util.CallApi(http.MethodPost, url, parameters, resp, 10, false)
	if err == nil && resp.ErrCode >= "400" {
		err = fmt.Errorf("%s %s", resp.ErrCode, resp.ErrMsg)
	}
	if err == nil && resp.Result == nil {
		err = errors.New("result is missing in ListInstanceStateConfigurations response")
	}
	return resp, err
}

func GetTemplate(templateName, templateVersion string) (*GetTemplateResponse, error) {
	url := util.GetTemplateService()
	parameters := map[string]interface{}{
		"templateName": templateName,
	}
	if len(templateVersion) > 0 {
		parameters["templateVersion"] = templateVersion
	}
	resp := &GetTemplateResponse{}
	err := util.CallApi(http.MethodPost, url, parameters, resp, 10, true)
	if err == nil && resp.ErrCode >= "400" {
		err = fmt.Errorf("%s %s", resp.ErrCode, resp.ErrMsg)
	}
	if err == nil && resp.Result == nil {
		err = errors.New("result is missing in GetTemplate response")
	}
	return resp, err
}

func PutInstanceStateReport(stateConfigurationId, status, extraInfo, mode, clientToken string) error {
	reportTime := timetool.UtcNowStr()
	url := util.GetPutInstanceStateReportService()
	parameters := map[string]interface{}{
		"stateConfigurationId": stateConfigurationId,
		"reportTime":           reportTime,
		"status":               status,
		"mode":                 mode,
	}
	if len(extraInfo) > 0 {
		parameters["extraInfo"] = extraInfo
	}
	if len(clientToken) > 0 {
		parameters["clientToken"] = clientToken
	}
	resp := &ApiResponse{}
	err := util.CallApi(http.MethodPost, url, parameters, resp, 10, false)
	if err == nil && resp.ErrCode >= "400" {
		err = fmt.Errorf("%s %s", resp.ErrCode, resp.ErrMsg)
	}
	return err
}
