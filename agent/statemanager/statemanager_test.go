package statemanager

import (
	"errors"
	"testing"
	"time"
	"fmt"
	"net/http"

	gomonkey "github.com/agiledragon/gomonkey/v2"
	"github.com/jarcoal/httpmock"
	"github.com/aliyun/aliyun_assist_client/agent/inventory/gatherers/instance"
	"github.com/aliyun/aliyun_assist_client/agent/inventory/model"
	"github.com/aliyun/aliyun_assist_client/agent/util/timetool"
	"github.com/stretchr/testify/assert"
	"github.com/aliyun/aliyun_assist_client/agent/statemanager/resources"
	"github.com/aliyun/aliyun_assist_client/agent/clientreport"
	"github.com/aliyun/aliyun_assist_client/agent/util"
	"github.com/aliyun/aliyun_assist_client/agent/taskengine/timermanager"
	"github.com/aliyun/aliyun_assist_client/internal/testutil"
	"github.com/aliyun/aliyun_assist_client/common/requester"
	"github.com/aliyun/aliyun_assist_client/thirdparty/sirupsen/logrus"
)

func TestGetMode1(t *testing.T) {
	config := StateConfiguration{
		StateConfigurationId: "sc-001",
		TemplateName:         "linux-host-file",
		TemplateVersion:      "1.0",
		Parameters:           `{"owner": "root"}`,
		ConfigureMode:        "ApplyAndMonitor",
		ScheduleType:         "cron",
		ScheduleExpression:   "0 0 */1 ? * *",
		// SuccessfulApplyTime  : "2020-10-20T00:00:00Z",
		DefinitionUpdateTime: "2020-10-21T00:00:00Z",
	}
	mode := getMode(config)
	assert.Equal(t, Apply, mode)
}

func TestGetMode2(t *testing.T) {
	config := StateConfiguration{
		StateConfigurationId: "sc-001",
		TemplateName:         "linux-host-file",
		TemplateVersion:      "1.0",
		Parameters:           `{"owner": "root"}`,
		ConfigureMode:        "ApplyAndMonitor",
		ScheduleType:         "cron",
		ScheduleExpression:   "0 0 */1 ? * *",
		SuccessfulApplyTime:  "2020-10-22T00:00:00Z",
		DefinitionUpdateTime: "2020-10-21T00:00:00Z",
	}
	mode := getMode(config)
	assert.Equal(t, Monitor, mode)
}

func TestGetMode3(t *testing.T) {
	config := StateConfiguration{
		StateConfigurationId: "sc-001",
		TemplateName:         "linux-host-file",
		TemplateVersion:      "1.0",
		Parameters:           `{"owner": "root"}`,
		ConfigureMode:        "ApplyAndMonitor",
		ScheduleType:         "cron",
		ScheduleExpression:   "0 0 */1 ? * *",
		SuccessfulApplyTime:  "2020-10-22T00:00:00Z",
		DefinitionUpdateTime: "2020-10-23T00:00:00Z",
	}
	mode := getMode(config)
	assert.Equal(t, Apply, mode)
}

func Test_refreshStateConfigs(t *testing.T) {
	timermanager.InitTimerManager()
	guard_clientReport := gomonkey.ApplyFunc(clientreport.SendReport, func(report clientreport.ClientReport) (string, error) {
		fmt.Println(report)
		panic(report)
})
	defer guard_clientReport.Reset()

	tests := []struct {
		name string
	}{
		{
			name: "instanceInfoFail",
		},
		{
			name: "ListInstanceStateConfigurationsError",
		},
		{
			name: "normal",
		},
	}

	guard := gomonkey.ApplyFunc(LoadConfigCache, func() (r *ListInstanceStateConfigurationsResult, err error) {
		r = &ListInstanceStateConfigurationsResult{
			Checkpoint: timetool.ApiTimeFormat(time.Now().Add(time.Duration(-2) * time.Minute)),
		}
		err = nil
		return
	})
	defer guard.Reset()
	instanceInfo := model.InstanceInformation{
		AgentName:       "agent",
		AgentVersion:    "version",
		ComputerName:    "computername",
		PlatformName:    "platformname",
		PlatformType:    "platformtype",
		PlatformVersion: "platformversion",
		InstanceId:      "instanceid",
		IpAddress:       "ip",
		ResourceType:    "resourcetype",
		RamRole:         "ramrole",
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.name == "instanceInfoFail" {
				guard := gomonkey.ApplyFunc(instance.GetInstanceInfo, func() (*model.InstanceInformation, error) {
					return nil, errors.New("some error")
				})
				defer guard.Reset()
			} else if tt.name == "ListInstanceStateConfigurationsError" {
				guard := gomonkey.ApplyFunc(ListInstanceStateConfigurations, func(lastCheckpoint, agentName, agentVersion, computerName,
					platformName, platformType, platformVersion, ipAddress, ramRole string) (*ListInstanceStateConfigurationsResp, error) {
					return &ListInstanceStateConfigurationsResp{
						ApiResponse: ApiResponse{
							ErrCode: "ServiceNotSupported",
						},
					}, errors.New("some error")
				})
				defer guard.Reset()
				guard_1 := gomonkey.ApplyFunc(instance.GetInstanceInfo, func() (*model.InstanceInformation, error) {
					return &instanceInfo, nil
				})
				defer guard_1.Reset()
			} else if tt.name == "normal" {
				guard := gomonkey.ApplyFunc(ListInstanceStateConfigurations, func(lastCheckpoint, agentName, agentVersion, computerName,
					platformName, platformType, platformVersion, ipAddress, ramRole string) (*ListInstanceStateConfigurationsResp, error) {
					return &ListInstanceStateConfigurationsResp{
						ApiResponse: ApiResponse{
							ErrCode: "ServiceNotSupported",
						},
						Result: &ListInstanceStateConfigurationsResult{
							Changed: true,
						},
					}, nil
				})
				defer guard.Reset()
				guard_1 := gomonkey.ApplyFunc(instance.GetInstanceInfo, func() (*model.InstanceInformation, error) {
					return &instanceInfo, nil
				})
				defer guard_1.Reset()
			}
			refreshStateConfigs()
		})
	}
}

func Test_enforce(t *testing.T) {
	stateConfig := StateConfiguration{
		ConfigureMode: ApplyOnly,
	}
	type args struct {
		config StateConfiguration
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "skip",
			args: args{
				config: stateConfig,
			},
			wantErr: false,
		},
		{
			name: "gettemplateError",
			args: args{
				config: stateConfig,
			},
			wantErr: true,
		},
		{
			name: "normalApply",
			args: args{
				config: stateConfig,
			},
			wantErr: false,
		},
		{
			name: "normalMonitor",
			args: args{
				config: stateConfig,
			},
			wantErr: false,
		},
	}

	guard_transport := gomonkey.ApplyFunc(requester.GetHTTPTransport, func(logrus.FieldLogger) *http.Transport {
		transport, _ := http.DefaultTransport.(*http.Transport)
		return transport
	})
	defer guard_transport.Reset()

	httpmock.Activate()
	util.NilRequest.Set()
	defer util.NilRequest.Clear()
	defer httpmock.DeactivateAndReset()
	const mockRegion = "cn-test100"
	testutil.MockMetaServer(mockRegion)

	httpmock.RegisterResponder("POST",
		fmt.Sprintf("https://%s.axt.aliyun.com/luban/api/instance/put_instance_state_report", mockRegion),
		func(h *http.Request) (*http.Response, error) {
			return httpmock.NewStringResponse(200, "success"), nil
		})

	guard := gomonkey.ApplyFunc(LoadTemplateCache, func(string, string) (content []byte, err error) {
		return nil, errors.New("some error")
	})
	defer guard.Reset()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.name == "skip" {
				tt.args.config.ConfigureMode = "unknown"
			} else if tt.name == "gettemplateError" {
				guard := gomonkey.ApplyFunc(GetTemplate, func(string, string) (*GetTemplateResponse, error) {
					return nil, errors.New("some error")
				})
				defer guard.Reset()
			} else if tt.name == "normalApply" || tt.name == "normalMonitor" {
				guard := gomonkey.ApplyFunc(GetTemplate, func(string, string) (*GetTemplateResponse, error) {
					return &GetTemplateResponse{
						ApiResponse: ApiResponse{},
						Result: &GetTemplateResult{
							Content: "content",
						},
					}, nil
				})
				defer guard.Reset()
				state := StateDef{
					ResourceType: "ACS:Inventory",
					Properties: make(map[string]interface{}),
				}
				guard_1 := gomonkey.ApplyFunc(ParseResourceState, func([]byte, string) ([]resources.ResourceState, error) {
					resourceStates := []resources.ResourceState{}
					resourceState, _ := NewResourceState(state)
					resourceStates = append(resourceStates, resourceState)
					return resourceStates, nil
				})
				defer guard_1.Reset()
				if tt.name == "normalMonitor" {
					tt.args.config.ConfigureMode = ApplyAndMonitor
				}
			}
			if err := enforce(tt.args.config); (err != nil) != tt.wantErr {
				t.Errorf("enforce() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
