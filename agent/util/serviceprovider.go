package util

// Copyright (c) 2017-2018 Alibaba Group Holding Limited.

func GetUpdateService() string {
	url := "https://" + GetServerHost()
	url += "/luban/api/v1/update/update_check"
	return url
}

func GetConnectDetectService() string {
	url := "https://" + GetServerHost()
	url += "/luban/api/connection_detect"
	return url
}

//New version 1.0
func GetInvalidTaskService() string {
	url := "https://" + GetServerHost()
	url += "/luban/api/v1/task/invalid"
	return url
}

func GetFetchTaskListService() string {
	url := "https://" + GetServerHost()
	url += "/luban/api/v1/task/list"
	return url
}

func GetVerifiedTaskService() string {
	url := "https://" + GetServerHost()
	url += "/luban/api/v1/task/verified"
	return url
}

func GetFetchSessionTaskListService() string {
	url := "https://" + GetServerHost()
	url += "/luban/api/v1/session/list"
	return url
}

func GetRunningOutputService() string {
	url := "https://" + GetServerHost()
	url += "/luban/api/v1/task/running"
	return url
}

func GetFinishOutputService() string {
	url := "https://" + GetServerHost()
	url += "/luban/api/v1/task/finish"
	return url
}

func GetSessionStatusService() string {
	url := "https://" + GetServerHost()
	url += "/luban/api/v1/session/status"
	return url
}

func GetStoppedOutputService() string {
	url := "https://" + GetServerHost()
	url += "/luban/api/v1/task/stopped"
	return url
}

func GetTimeoutOutputService() string {
	url := "https://" + GetServerHost()
	url += "/luban/api/v1/task/timeout"
	return url
}

func GetErrorOutputService() string {
	url := "https://" + GetServerHost()
	url += "/luban/api/v1/task/error"
	return url
}

func GetPingService() string {
	url := "https://" + GetServerHost()
	url += "/luban/api/heart-beat"
	return url
}

func GetGshellCheckService() string {
	url := "https://" + GetServerHost()
	url += "/luban/api/gshell"
	return url
}

func GetPluginListService() string {
	url := "https://" + GetServerHost()
	url += "/luban/api/v1/plugin/list"
	return url
}

func GetClientReportService() string {
	url := "https://" + GetServerHost()
	url += "/luban/api/v1/exception/client_report"
	return url
}

func GetStateConfigService() string {
	url := "https://" + GetServerHost()
	url += "/luban/api/instance/list_instance_state_configurations"
	return url
}

func GetTemplateService() string {
	url := "https://" + GetServerHost()
	url += "/luban/api/instance/get_template"
	return url
}

func GetPutInventoryService() string {
	url := "https://" + GetServerHost()
	url += "/luban/api/instance/put_inventory"
	return url
}

func GetPutInstanceStateReportService() string {
	url := "https://" + GetServerHost()
	url += "/luban/api/instance/put_instance_state_report"
	return url
}
func GetRegisterService() string {
	url := "https://" + GetServerHost()
	url += "/luban/api/instance/register"
	return url
}

func GetDeRegisterService() string {
	url := "https://" + GetServerHost()
	url += "/luban/api/instance/register"
	return url
}
