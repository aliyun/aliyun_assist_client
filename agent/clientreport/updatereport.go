package clientreport

import "encoding/json"

const (
	_reportTypePrefix = "AgentUpdateFailure:"
)

type UpdateFailure struct {
	UpdateInfo interface{} `json:"updateInfo"`
	FailureContext map[string]interface{} `json:"failureContext"`
	ErrorMessage string `json:"errorMessage"`
}

func ReportUpdateFailure(failureType string, failure UpdateFailure) (string, error) {
	failureJSONBytes, err := json.Marshal(failure)
	if err != nil {
		return "", err
	}

	report := ClientReport{
		ReportType: _reportTypePrefix + failureType,
		Info:       string(failureJSONBytes),
	}
	return SendReport(report)
}


