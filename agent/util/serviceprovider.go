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

func GetRunningOutputService() string {
	url := "https://" + GetServerHost()
	url += "/luban/api/v1/task/running"
	return url
}

func GetFinishOutputService() string {
	url := "https://" + GetServerHost()
	url += "/luban/api/v1/task/finish"
	return url;
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

func  GetGshellCheckService() string {
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
