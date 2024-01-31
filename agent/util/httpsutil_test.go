package util

import (
	"crypto/x509"
	"reflect"
	"testing"

	gomonkey "github.com/agiledragon/gomonkey/v2"
	"github.com/aliyun/aliyun_assist_client/common/apiserver"
	"github.com/aliyun/aliyun_assist_client/agent/log"
	"github.com/aliyun/aliyun_assist_client/thirdparty/sirupsen/logrus"
	"github.com/kirinlabs/HttpRequest"
	"github.com/stretchr/testify/assert"
)

var (
	isCertRight = false
)

func TestHttpGet(t *testing.T) {
	// guard_IsSystemdLinux := gomonkey.ApplyFunc(util.IsSystemdLinux, func() bool { return true })
	// guard_metricsReportEvent := gomonkey.ApplyMethod(reflect.TypeOf(m), "ReportEvent", func(event *metrics.MetricsEvent) {
	var r *HttpRequest.Response
	guard_ResponseClose := gomonkey.ApplyMethod(reflect.TypeOf(r), "Close", func() error { return nil })
	defer guard_ResponseClose.Reset()
	guard_ResponseContent := gomonkey.ApplyMethod(reflect.TypeOf(r), "Content", func() (string, error) { return "ok", nil })
	defer guard_ResponseContent.Reset()
	guard_ResponseStatusCode := gomonkey.ApplyMethod(reflect.TypeOf(r), "StatusCode", func() int { return 200 })
	defer guard_ResponseStatusCode.Reset()

	var req *HttpRequest.Request
	guard_httpGet := gomonkey.ApplyMethod(reflect.TypeOf(req), "Get", func (r *HttpRequest.Request, url string, data ...interface{}) (*HttpRequest.Response, error) {
		if isCertRight {
			return &HttpRequest.Response{}, nil
		} else {
			return nil, x509.UnknownAuthorityError{}
		}
	})
	defer guard_httpGet.Reset()
	guard_httpPost := gomonkey.ApplyMethod(reflect.TypeOf(req), "Post", func (r *HttpRequest.Request, url string, data ...interface{}) (*HttpRequest.Response, error) {
		if isCertRight {
			return &HttpRequest.Response{}, nil
		} else {
			return nil, x509.UnknownAuthorityError{}
		}
	})
	defer guard_httpPost.Reset()

	var p *apiserver.ExternalExecutableProvider
	guard_ExternalExecutableProviderCACertificate := gomonkey.ApplyMethod(reflect.TypeOf(p), 
		"CACertificate", func(p *apiserver.ExternalExecutableProvider, logger logrus.FieldLogger, refresh bool) ([]byte, error) {
			if refresh {
				isCertRight = !isCertRight
			}
			return []byte("abc"), nil
		})
	defer guard_ExternalExecutableProviderCACertificate.Reset()
	guard_ExternalExecutableProviderName := gomonkey.ApplyMethod(reflect.TypeOf(p), 
		"Name", func(p *apiserver.ExternalExecutableProvider) string {
			return "test-provider"
		})
	defer guard_ExternalExecutableProviderName.Reset()
	guard_ExternalExecutableProviderServerDomain := gomonkey.ApplyMethod(reflect.TypeOf(p), 
		"ServerDomain", func(p *apiserver.ExternalExecutableProvider) (string, error) {
			return "test-domain", nil
		})
	defer guard_ExternalExecutableProviderServerDomain.Reset()
	guard_ExternalExecutableProviderExtraHTTPHeaders := gomonkey.ApplyMethod(reflect.TypeOf(p), 
		"ExtraHTTPHeaders", func(p *apiserver.ExternalExecutableProvider) (map[string]string, error) {
			return make(map[string]string), nil
		})
	defer guard_ExternalExecutableProviderExtraHTTPHeaders.Reset()
	guard_ExternalExecutableProviderRegionId := gomonkey.ApplyMethod(reflect.TypeOf(p), 
		"RegionId", func(p *apiserver.ExternalExecutableProvider) (string, error) {
			return "cn-test", nil
		})
	defer guard_ExternalExecutableProviderRegionId.Reset()

	for i:=0; i<5; i += 1 {
		log.GetLogger().Infof("---------------------------HTTPGet %d---------------------------", i)
		err, content := HttpGet("https://abc.abc")
		assert.Equal(t, nil, err)
		assert.Equal(t, "ok", content)
	}
	for i:=0; i<5; i += 1 {
		log.GetLogger().Infof("---------------------------HTTP Post%d---------------------------", i)
		content, err := HttpPost("https://abc.abc", "data", "contentType")
		assert.Equal(t, nil, err)
		assert.Equal(t, "ok", content)
	}
}