package util

import "os"

// Copyright (c) 2017-2023 Alibaba Group Holding Limited.

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

func GetMetricsService() string {
	url := "https://" + GetServerHost()
	url += "/luban/api/metrics"
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

// GetPingService returns heart-beat API but without the scheme part, unlike
// other API address provider function
func GetPingService() string {
	url := GetServerHost()
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

func GetPluginHealthService() string {
	url := "https://" + GetServerHost()
	url += "/luban/api/v1/plugin/report_status"
	return url
}

func GetPluginUpdateCheckService() string {
	url := "https://" + GetServerHost()
	url += "/luban/api/v2/plugin/update_check"
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

func GetDeRegisterService() string {
	url := "https://" + GetServerHost()
	url += "/luban/api/instance/deregister"
	return url
}

func GetRegisterService(region, networkmode string) string {
	if IsSelfHosted() {
		host := os.Getenv("ALIYUN_ASSIST_SERVER_HOST")
		return "https://" + host + "/luban/api/instance/register"
	}
	domain := HYBRID_DOMAIN
	if networkmode == "vpc" {
		domain = HYBRID_DOMAIN_VPC
	}
	url := "https://" + region + domain + "/luban/api/instance/register"	
	return url
}
