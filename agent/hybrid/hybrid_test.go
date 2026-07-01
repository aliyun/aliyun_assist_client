package hybrid

import (
	"bytes"
	"errors"
	"net/http"
	"reflect"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"

	"github.com/aliyun/aliyun_assist_client/agent/hybrid/instance"
	"github.com/aliyun/aliyun_assist_client/agent/metrics"
	"github.com/aliyun/aliyun_assist_client/agent/util"
	"github.com/aliyun/aliyun_assist_client/agent/util/serviceutil"
	"github.com/aliyun/aliyun_assist_client/common/apiserver"
	"github.com/aliyun/aliyun_assist_client/common/requester"
	"github.com/aliyun/aliyun_assist_client/internal/testutil"
	"github.com/aliyun/aliyun_assist_client/thirdparty/sirupsen/logrus"
)

func TestKeyPair(t *testing.T) {
	var pub, pri bytes.Buffer
	err := genRsaKey(&pub, &pri)
	assert.Equal(t, err, nil)
	assert.True(t, len(pub.String()) > 200)
	assert.True(t, len(pub.String()) > 200)
}

func TestRegister(t *testing.T) {
	defer gomonkey.ApplyFunc(requester.GetHTTPTransport, func(logrus.FieldLogger) *http.Transport {
		transport, _ := http.DefaultTransport.(*http.Transport)
		return transport
	}).Reset()

	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	requester.NilTransport.Set()
	defer requester.NilTransport.Clear()
	const region = "cn-test100"
	testutil.MockMetaServer(region)

	defer gomonkey.ApplyFunc(util.GetServerHost, func() string {
		return region + apiserver.HybridDomainFirst
	}).Reset()
	var m *metrics.MetricsEvent
	defer gomonkey.ApplyMethod(reflect.TypeOf(m), "ReportEvent", func(me *metrics.MetricsEvent) {}).Reset()

	var restartAgentServiceCalled bool
	defer gomonkey.ApplyFunc(serviceutil.RestartAgentService, func(logrus.FieldLogger) {
		restartAgentServiceCalled = true
	}).Reset()

	url := "https://" + region + apiserver.HybridDomainFirst + "/luban/api/instance/register"
	httpmock.RegisterResponder("POST", url,
		httpmock.NewStringResponder(200, `{"code":200,"instanceId":"xx-123"}`))
	url = "https://" + region + util.HYBRID_DOMAIN_VPC + "/luban/api/instance/register"
	httpmock.RegisterResponder("POST", url,
		httpmock.NewStringResponder(200, `{"code":200,"instanceId":"xx-123"}`))
	unregister_url := "https://" + util.GetServerHost() + "/luban/api/instance/deregister"
	httpmock.RegisterResponder("POST", unregister_url,
		httpmock.NewStringResponder(200, `{"code":200}`))

	// Register and need restart
	UnRegister(false)
	restartAgentServiceCalled = false
	needRestart := true
	ret := Register(region, "test_code", "test_id", "test_machine", "vpc", needRestart, nil, "", false)
	assert.True(t, ret)
	assert.True(t, instance.IsHybrid())
	assert.True(t, restartAgentServiceCalled)

	assert.Equal(t, instance.ReadInstanceId(), "xx-123")
	assert.Equal(t, instance.ReadRegionId(), region)

	// Deregister and need restart
	restartAgentServiceCalled = false
	needRestart = true
	UnRegister(needRestart)
	assert.False(t, instance.IsHybrid())
	assert.True(t, restartAgentServiceCalled)

	// Register and no need restart
	UnRegister(false)
	restartAgentServiceCalled = false
	needRestart = false
	ret = Register(region, "test_code", "test_id", "test_machine", "vpc", needRestart, nil, "", false)
	assert.True(t, ret)
	assert.True(t, instance.IsHybrid())
	assert.False(t, restartAgentServiceCalled)

	assert.Equal(t, instance.ReadInstanceId(), "xx-123")
	assert.Equal(t, instance.ReadRegionId(), region)

	// Deregister and no need restart
	restartAgentServiceCalled = false
	needRestart = false
	UnRegister(needRestart)
	assert.False(t, instance.IsHybrid())
	assert.False(t, restartAgentServiceCalled)
}

type mockIdentifier struct {
	generate func() (string, error)
}

func (m *mockIdentifier) Name() string {
	return "mock"
}

func (m *mockIdentifier) Generate() (string, error) {
	return m.generate()
}

type mockIdentifyRefresher struct {
	generate func() (string, error)
	refresh  func() (string, error)
}

func (m *mockIdentifyRefresher) Name() string {
	return "mockRefresher"
}

func (m *mockIdentifyRefresher) Generate() (string, error) {
	return m.generate()
}

func (m *mockIdentifyRefresher) Refresh() (string, error) {
	return m.refresh()
}

func TestTryRegister(t *testing.T) {
	t.Run("NormalCase", func(t *testing.T) {
		doRegisterCounter := 0
		saveCalled := false

		mockIdent := &mockIdentifier{
			generate: func() (string, error) {
				return "validFingerprint", nil
			},
		}
		patches := gomonkey.NewPatches()
		defer patches.Reset()
		patches.ApplyFunc(doRegister, func(*RegisterInfo, string, map[string]string) (registerResponse, error) {
			doRegisterCounter++

			return registerResponse{Code: 200, InstanceId: "test-123"}, nil
		})
		patches.ApplyFunc(instance.SaveInstanceInfo, func(string, string, string, string, string, string, string) { saveCalled = true })

		instanceId, err := tryRegister(&RegisterInfo{}, mockIdent, "vpc", nil, "", "")
		assert.NoError(t, err)
		assert.Equal(t, "test-123", instanceId)
		assert.Equal(t, 1, doRegisterCounter)
		assert.True(t, saveCalled)
	})

	t.Run("DisableECSRegistration", func(t *testing.T) {
		doRegisterCounter := 0
		saveCalled := false

		mockIdent := &mockIdentifier{
			generate: func() (string, error) {
				return "validFingerprint", nil
			},
		}
		patches := gomonkey.NewPatches()
		defer patches.Reset()
		patches.ApplyFunc(doRegister, func(*RegisterInfo, string, map[string]string) (registerResponse, error) {
			doRegisterCounter++

			return registerResponse{Code: 403, ErrCode: errCodeEcsRegDisabled}, nil
		})
		patches.ApplyFunc(instance.SaveInstanceInfo, func(string, string, string, string, string, string, string) { saveCalled = true })

		_, err := tryRegister(&RegisterInfo{}, mockIdent, "vpc", nil, "", "")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "The ECS instance is forbidden from registering")
		assert.Equal(t, 1, doRegisterCounter)
		assert.False(t, saveCalled)
	})

	t.Run("DuplicateFingerprintButRefresh", func(t *testing.T) {
		doRegisterCounter := 0
		saveCalled := false

		mockRefresher := &mockIdentifyRefresher{
			generate: func() (string, error) {
				return "duplicateFingerprint", nil
			},
			refresh: func() (string, error) {
				return "newFingerprint", nil
			},
		}
		patches := gomonkey.NewPatches()
		defer patches.Reset()
		patches.ApplyFunc(doRegister, func(info *RegisterInfo, _ string, _ map[string]string) (registerResponse, error) {
			doRegisterCounter++

			if info.Fingerprint == "duplicateFingerprint" {
				return registerResponse{Code: 409, ErrCode: errCodeFingerprintDuplicate}, nil
			}

			return registerResponse{Code: 200, InstanceId: "test-123"}, nil
		})
		patches.ApplyFunc(instance.SaveInstanceInfo, func(string, string, string, string, string, string, string) { saveCalled = true })

		instanceId, err := tryRegister(&RegisterInfo{}, mockRefresher, "vpc", nil, "", "")

		assert.NoError(t, err)
		assert.Equal(t, "test-123", instanceId)
		assert.Equal(t, 2, doRegisterCounter)
		assert.True(t, saveCalled)
	})

	t.Run("DuplicateFingerprintWithoutRefresh", func(t *testing.T) {
		doRegisterCounter := 0
		saveCalled := false

		mockIdent := &mockIdentifier{
			generate: func() (string, error) {
				return "duplicateFingerprint", nil
			},
		}
		patches := gomonkey.NewPatches()
		defer patches.Reset()
		patches.ApplyFunc(doRegister, func(*RegisterInfo, string, map[string]string) (registerResponse, error) {
			doRegisterCounter++

			return registerResponse{Code: 409, ErrCode: errCodeFingerprintDuplicate}, nil
		})
		patches.ApplyFunc(instance.SaveInstanceInfo, func(string, string, string, string, string, string, string) { saveCalled = true })

		_, err := tryRegister(&RegisterInfo{}, mockIdent, "vpc", nil, "", "")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Register failed of duplicate machine")
		assert.Equal(t, 1, doRegisterCounter)
		assert.False(t, saveCalled)
	})

	t.Run("RefreshedFingerprintButDeclined", func(t *testing.T) {
		doRegisterCounter := 0
		saveCalled := false

		mockRefresher := &mockIdentifyRefresher{
			generate: func() (string, error) {
				return "duplicateFingerprint", nil
			},
			refresh: func() (string, error) {
				return "newFingerprint", nil
			},
		}
		patches := gomonkey.NewPatches()
		defer patches.Reset()
		patches.ApplyFunc(doRegister, func(info *RegisterInfo, _ string, _ map[string]string) (registerResponse, error) {
			doRegisterCounter++

			if info.Fingerprint == "duplicateFingerprint" {
				return registerResponse{Code: 409, ErrCode: errCodeFingerprintDuplicate}, nil
			}

			return registerResponse{Code: 418, ErrCode: "I'm a teapot"}, nil
		})
		patches.ApplyFunc(instance.SaveInstanceInfo, func(string, string, string, string, string, string, string) { saveCalled = true })

		_, err := tryRegister(&RegisterInfo{}, mockRefresher, "vpc", nil, "", "")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Register failed")
		assert.Contains(t, err.Error(), "I'm a teapot")
		assert.Equal(t, 2, doRegisterCounter)
		assert.False(t, saveCalled)
	})

	t.Run("GenerateFailure", func(t *testing.T) {
		doRegisterCounter := 0
		saveCalled := false

		mockIdent := &mockIdentifier{
			generate: func() (string, error) {
				return "", errors.New("concrete failure")
			},
		}
		patches := gomonkey.NewPatches()
		defer patches.Reset()
		patches.ApplyFunc(doRegister, func(*RegisterInfo, string, map[string]string) (registerResponse, error) {
			doRegisterCounter++

			return registerResponse{Code: 200, InstanceId: "test-123"}, nil
		})
		patches.ApplyFunc(instance.SaveInstanceInfo, func(string, string, string, string, string, string, string) { saveCalled = true })

		_, err := tryRegister(&RegisterInfo{}, mockIdent, "vpc", nil, "", "")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "generate fingerprint failed")
		assert.Contains(t, err.Error(), "concrete failure")
		assert.Equal(t, 0, doRegisterCounter)
		assert.False(t, saveCalled)
	})

	t.Run("RefreshFailure", func(t *testing.T) {
		doRegisterCounter := 0
		saveCalled := false

		mockRefresher := &mockIdentifyRefresher{
			generate: func() (string, error) {
				return "duplicateFingerprint", nil
			},
			refresh: func() (string, error) {
				return "", errors.New("concrete failure")
			},
		}
		patches := gomonkey.NewPatches()
		defer patches.Reset()
		patches.ApplyFunc(doRegister, func(*RegisterInfo, string, map[string]string) (registerResponse, error) {
			doRegisterCounter++

			return registerResponse{Code: 409, ErrCode: errCodeFingerprintDuplicate}, nil
		})
		patches.ApplyFunc(instance.SaveInstanceInfo, func(string, string, string, string, string, string, string) { saveCalled = true })

		_, err := tryRegister(&RegisterInfo{}, mockRefresher, "vpc", nil, "", "")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Regenerate fingerprint failed")
		assert.Contains(t, err.Error(), "concrete failure")
		assert.Equal(t, 1, doRegisterCounter)
		assert.False(t, saveCalled)
	})

	t.Run("DoRegisterError", func(t *testing.T) {
		doRegisterCounter := 0
		refreshCalled := false
		saveCalled := false

		mockRefresher := &mockIdentifyRefresher{
			generate: func() (string, error) {
				return "validFingerprint", nil
			},
			refresh: func() (string, error) {
				refreshCalled = true
				return "", errors.New("SHOULD NOT REACH HERE")
			},
		}
		patches := gomonkey.NewPatches()
		defer patches.Reset()
		patches.ApplyFunc(doRegister, func(*RegisterInfo, string, map[string]string) (registerResponse, error) {
			doRegisterCounter++

			return registerResponse{}, errors.New("doRegister failure")
		})
		patches.ApplyFunc(instance.SaveInstanceInfo, func(string, string, string, string, string, string, string) { saveCalled = true })

		_, err := tryRegister(&RegisterInfo{}, mockRefresher, "vpc", nil, "", "")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Register failed")
		assert.Contains(t, err.Error(), "doRegister failure")
		assert.Equal(t, 1, doRegisterCounter)
		assert.False(t, refreshCalled)
		assert.False(t, saveCalled)
	})

	t.Run("DoRegisterUnexpectedCode", func(t *testing.T) {
		doRegisterCounter := 0
		refreshCalled := false
		saveCalled := false

		mockRefresher := &mockIdentifyRefresher{
			generate: func() (string, error) {
				return "validFingerprint", nil
			},
			refresh: func() (string, error) {
				refreshCalled = true
				return "", errors.New("SHOULD NOT REACH HERE")
			},
		}
		patches := gomonkey.NewPatches()
		defer patches.Reset()
		patches.ApplyFunc(doRegister, func(*RegisterInfo, string, map[string]string) (registerResponse, error) {
			doRegisterCounter++

			return registerResponse{Code: 418, ErrCode: "I'm a teapot"}, nil
		})
		patches.ApplyFunc(instance.SaveInstanceInfo, func(string, string, string, string, string, string, string) { saveCalled = true })

		_, err := tryRegister(&RegisterInfo{}, mockRefresher, "vpc", nil, "", "")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Register failed")
		assert.Contains(t, err.Error(), "I'm a teapot")
		assert.Equal(t, 1, doRegisterCounter)
		assert.False(t, refreshCalled)
		assert.False(t, saveCalled)
	})
}
