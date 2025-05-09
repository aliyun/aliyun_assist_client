package requester

import (
	"testing"

	"github.com/aliyun/aliyun_assist_client/thirdparty/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestNilTransport(t *testing.T) {
	transport := GetHTTPTransport(logrus.New())
	assert.NotNil(t, transport)
	transport = GetProxiedHTTPTransport(logrus.New())
	assert.NotNil(t, transport)

	NilTransport.Set()
	defer NilTransport.Clear()
	transport = GetHTTPTransport(logrus.New())
	assert.Nil(t, transport)
	transport = GetProxiedHTTPTransport(logrus.New())
	assert.Nil(t, transport)
}
