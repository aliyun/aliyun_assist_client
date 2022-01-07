package statemanager

import (
	"errors"
	"testing"
	"time"

	"bou.ke/monkey"
	"github.com/aliyun/aliyun_assist_client/agent/inventory/gatherers/instance"
	"github.com/aliyun/aliyun_assist_client/agent/inventory/model"
	"github.com/aliyun/aliyun_assist_client/agent/util/timetool"
	"github.com/stretchr/testify/assert"
	"github.com/aliyun/aliyun_assist_client/agent/statemanager/resources"
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

	guard := monkey.Patch(LoadConfigCache, func() (r *ListInstanceStateConfigurationsResult, err error) {
		r = &ListInstanceStateConfigurationsResult{
			Checkpoint: timetool.ApiTimeFormat(time.Now().Add(time.Duration(-2) * time.Minute)),
		}
		err = nil
		return
	})
	defer guard.Unpatch()
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
				guard := monkey.Patch(instance.GetInstanceInfo, func() (*model.InstanceInformation, error) {
					return nil, errors.New("some error")
				})
				defer guard.Unpatch()
			} else if tt.name == "ListInstanceStateConfigurationsError" {
				guard := monkey.Patch(ListInstanceStateConfigurations, func(lastCheckpoint, agentName, agentVersion, computerName,
					platformName, platformType, platformVersion, ipAddress, ramRole string) (*ListInstanceStateConfigurationsResp, error) {
					return &ListInstanceStateConfigurationsResp{
						ApiResponse: ApiResponse{
							ErrCode: "ServiceNotSupported",
						},
					}, errors.New("some error")
				})
				defer guard.Unpatch()
				guard_1 := monkey.Patch(instance.GetInstanceInfo, func() (*model.InstanceInformation, error) {
					return &instanceInfo, nil
				})
				defer guard_1.Unpatch()
			} else if tt.name == "normal" {
				guard := monkey.Patch(ListInstanceStateConfigurations, func(lastCheckpoint, agentName, agentVersion, computerName,
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
				defer guard.Unpatch()
				guard_1 := monkey.Patch(instance.GetInstanceInfo, func() (*model.InstanceInformation, error) {
					return &instanceInfo, nil
				})
				defer guard_1.Unpatch()
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
	guard := monkey.Patch(LoadTemplateCache, func(string, string) (content []byte, err error) {
		return nil, errors.New("some error")
	})
	defer guard.Unpatch()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.name == "skip" {
				tt.args.config.ConfigureMode = "unknown"
			} else if tt.name == "gettemplateError" {
				guard := monkey.Patch(GetTemplate, func(string, string) (*GetTemplateResponse, error) {
					return nil, errors.New("some error")
				})
				defer guard.Unpatch()
			} else if tt.name == "normalApply" || tt.name == "normalMonitor" {
				guard := monkey.Patch(GetTemplate, func(string, string) (*GetTemplateResponse, error) {
					return &GetTemplateResponse{
						ApiResponse: ApiResponse{},
						Result: &GetTemplateResult{
							Content: "content",
						},
					}, nil
				})
				defer guard.Unpatch()
				state := StateDef{
					ResourceType: "ACS:Inventory",
					Properties: make(map[string]interface{}),
				}
				guard_1 := monkey.Patch(ParseResourceState, func([]byte, string) ([]resources.ResourceState, error) {
					resourceStates := []resources.ResourceState{}
					resourceState, _ := NewResourceState(state)
					resourceStates = append(resourceStates, resourceState)
					return resourceStates, nil
				})
				defer guard_1.Unpatch()
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
