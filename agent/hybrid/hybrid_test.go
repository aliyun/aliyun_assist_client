package hybrid

import (
	"bytes"
	"reflect"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"

	"github.com/aliyun/aliyun_assist_client/agent/hybrid/instance"
	"github.com/aliyun/aliyun_assist_client/agent/metrics"
	"github.com/aliyun/aliyun_assist_client/agent/util"
	"github.com/aliyun/aliyun_assist_client/internal/testutil"
)

func TestKeyPair(t *testing.T) {
    var pub,pri bytes.Buffer
	err := genRsaKey(&pub, &pri)
	assert.Equal(t, err, nil)
	assert.True(t,  len(pub.String()) > 200 )
	assert.True(t,  len(pub.String()) > 200 )
}

func TestRegister(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	util.NilRequest.Set()
	defer util.NilRequest.Clear()

	region := "cn-test100"

	testutil.MockMetaServer(region)

	guard_1 := gomonkey.ApplyFunc(util.GetServerHost, func() string {
		return region + util.HYBRID_DOMAIN
	} )
	defer guard_1.Reset()
	var m *metrics.MetricsEvent
	guard_4 := gomonkey.ApplyMethod(reflect.TypeOf(m), "ReportEvent", func(me *metrics.MetricsEvent) {})
	defer guard_4.Reset()

	url := "https://" + region + util.HYBRID_DOMAIN + "/luban/api/instance/register";
	httpmock.RegisterResponder("POST", url,
		httpmock.NewStringResponder(200, `{"code":200,"instanceId":"xx-123"}`))
	url = "https://" + region + util.HYBRID_DOMAIN_VPC + "/luban/api/instance/register";
	httpmock.RegisterResponder("POST", url,
		httpmock.NewStringResponder(200, `{"code":200,"instanceId":"xx-123"}`))
	unregister_url := "https://" + util.GetServerHost() + "/luban/api/instance/deregister";
	httpmock.RegisterResponder("POST", unregister_url,
		httpmock.NewStringResponder(200, `{"code":200}`))

	UnRegister(false)
	ret := Register(region, "test_code", "test_id", "test_machine", "vpc", false, nil)
	assert.True(t, ret)
	assert.True(t, instance.IsHybrid())

	assert.Equal(t, instance.ReadInstanceId(), "xx-123" )
	assert.Equal(t, instance.ReadRegionId(), region)

	UnRegister(false)
	assert.False(t, instance.IsHybrid())
	_, err := instance.ReadHardwareInfo()
	assert.Equal(t, nil, err)
}