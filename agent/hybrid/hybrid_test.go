package hybrid

import (
	"bytes"
	"github.com/aliyun/aliyun_assist_client/agent/util"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"testing"
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

	util.MockMetaServer(region)

	url := "https://" + region + util.HYBRID_DOMAIN + "/luban/api/instance/register";
	httpmock.RegisterResponder("POST", url,
		httpmock.NewStringResponder(200, `{"code":200,"instanceId":"xx-123"}`))
	url = "https://" + region + util.HYBRID_DOMAIN_VPC + "/luban/api/instance/register";
	httpmock.RegisterResponder("POST", url,
		httpmock.NewStringResponder(200, `{"code":200,"instanceId":"xx-123"}`))
	UnRegister(false)
	ret := Register(region, "test_code", "test_id", "test_machine", "vpc", false, nil)

	assert.True(t, ret)
	path,_ := util.GetHybridPath()
	path_instance_id := path + "/instance-id"
	content,_ := ioutil.ReadFile(path_instance_id)
	assert.Equal(t, string(content), "xx-123" )

	path_region_id := path + "/region-id"
	content,_ = ioutil.ReadFile(path_region_id)
	assert.Equal(t, string(content), region)

	unregister_url := "https://" + util.GetServerHost();
	unregister_url += "/luban/api/instance/deregister";
	httpmock.RegisterResponder("POST", unregister_url,
		httpmock.NewStringResponder(200, `{"code":200}`))

	UnRegister(false)

	ret = util.CheckFileIsExist(path_instance_id)
	assert.Equal(t, false, ret)
}