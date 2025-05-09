package hybrid

import (
	"bytes"
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
	ret := Register(region, "test_code", "test_id", "test_machine", "vpc", needRestart, nil)
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
	_, err := instance.ReadHardwareInfo()
	assert.Equal(t, nil, err)
	assert.True(t, restartAgentServiceCalled)

	// Register and no need restart
	UnRegister(false)
	restartAgentServiceCalled = false
	needRestart = false
	ret = Register(region, "test_code", "test_id", "test_machine", "vpc", needRestart, nil)
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
	_, err = instance.ReadHardwareInfo()
	assert.Equal(t, nil, err)
	assert.False(t, restartAgentServiceCalled)
}

