package util

import (
	"crypto/tls"
	"errors"
	"reflect"
	"testing"

	gomonkey "github.com/agiledragon/gomonkey/v2"
	"github.com/aliyun/aliyun_assist_client/agent/log"
	"github.com/aliyun/aliyun_assist_client/common/apiserver"
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
			return nil, &tls.CertificateVerificationError{}
		}
	})
	defer guard_httpGet.Reset()
	guard_httpPost := gomonkey.ApplyMethod(reflect.TypeOf(req), "Post", func (r *HttpRequest.Request, url string, data ...interface{}) (*HttpRequest.Response, error) {
		if isCertRight {
			return &HttpRequest.Response{}, nil
		} else {
			return nil, &tls.CertificateVerificationError{}
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

func TestHttpXxx(t *testing.T) {
	var (
		mockContent string
		mockStatusCode int
		mockReqError error

		reloadCert bool

		handlerContent string
		handlerStatusCode int
		handlerHttpErr error
	)
	SetHTTPPostErrHandler(func(httpResp *HttpRequest.Response, httpErr error) {
		if httpResp != nil {
			handlerContent, _ = httpResp.Content()
			handlerStatusCode = httpResp.StatusCode()
		}
		handlerHttpErr = httpErr
	})


	var r *HttpRequest.Response
	defer gomonkey.ApplyMethod(reflect.TypeOf(r), "Close", func() error { return nil }).Reset()
	defer gomonkey.ApplyMethod(reflect.TypeOf(r), "Content", func() (string, error) { return mockContent, nil }).Reset()
	defer gomonkey.ApplyMethod(reflect.TypeOf(r), "StatusCode", func() int { return mockStatusCode }).Reset()

	var req *HttpRequest.Request
	defer gomonkey.ApplyMethod(reflect.TypeOf(req), "Get", func (r *HttpRequest.Request, url string, data ...interface{}) (*HttpRequest.Response, error) {
		if mockReqError == nil {
			return &HttpRequest.Response{}, nil
		} else {
			return nil, mockReqError
		}
	}).Reset()
	defer gomonkey.ApplyMethod(reflect.TypeOf(req), "Post", func (r *HttpRequest.Request, url string, data ...interface{}) (*HttpRequest.Response, error) {
		if mockReqError == nil {
			return &HttpRequest.Response{}, nil
		} else {
			return nil, mockReqError
		}
	}).Reset()

	var p *apiserver.ExternalExecutableProvider
	defer gomonkey.ApplyMethod(reflect.TypeOf(p), 
		"CACertificate", func(p *apiserver.ExternalExecutableProvider, logger logrus.FieldLogger, refresh bool) ([]byte, error) {
			
			return []byte("abc"), nil
		}).Reset()
	defer gomonkey.ApplyMethod(reflect.TypeOf(p), 
		"Name", func(p *apiserver.ExternalExecutableProvider) string {
			return "test-provider"
		}).Reset()
	defer gomonkey.ApplyMethod(reflect.TypeOf(p), 
		"ServerDomain", func(p *apiserver.ExternalExecutableProvider) (string, error) {
			return "test-domain", nil
		}).Reset()
	defer gomonkey.ApplyMethod(reflect.TypeOf(p), 
		"ExtraHTTPHeaders", func(p *apiserver.ExternalExecutableProvider) (map[string]string, error) {
			return make(map[string]string), nil
		}).Reset()
	defer gomonkey.ApplyMethod(reflect.TypeOf(p), 
		"RegionId", func(p *apiserver.ExternalExecutableProvider) (string, error) {
			return "cn-test", nil
		}).Reset()


	testCases := []struct{
		name string
		url string
		data string
		contentType string
		method string

		expectErr error
		expectContent string
		expectStatusCode int
	}{
		// //////////////// GET ///////////////////
		{
			name: "get_ok_200",
			url: "https://abc.abc",
			method: "GET",

			expectErr: nil,
			expectContent: "ok",
			expectStatusCode: 200,
		},
		{
			name: "get_cert_err",
			url: "https://abc.abc",
			method: "GET",
			
			expectErr: &tls.CertificateVerificationError{},
		},
		// //////////////// POST ///////////////////
		{
			name: "post_ok_200",
			url: "https://abc.abc",
			method: "POST",

			expectErr: nil,
			expectContent: "ok",
			expectStatusCode: 200,
		},
		{
			name: "post_cert_err",
			url: "https://abc.abc",
			method: "POST",
			
			expectErr: &tls.CertificateVerificationError{},
		},
	}

	for _, tc := range testCases {
		reloadCert = false
		mockContent = tc.expectContent
		mockStatusCode = tc.expectStatusCode
		mockReqError = tc.expectErr

		t.Run(tc.name, func(t *testing.T) {
			if tc.method == "GET" {
				err, content := HttpGet(tc.url)
				assert.Equal(t, tc.expectErr, err)
				if tc.expectContent != "" {
					assert.Equal(t, tc.expectContent, content)
				}

				if errors.Is(err, &tls.CertificateVerificationError{}) {
					assert.True(t, reloadCert)
				}
			} else if tc.method == "POST" {
				content, err := HttpPost(tc.url, tc.data, tc.contentType)
				assert.Equal(t, tc.expectErr, err)
				if tc.expectContent != "" {
					assert.Equal(t, tc.expectContent, content)
				}

				if errors.Is(err, &tls.CertificateVerificationError{}) {
					assert.True(t, reloadCert)
				}
				if err != nil {
					assert.Equal(t, mockContent, handlerContent)
					assert.Equal(t, mockStatusCode, handlerStatusCode)
					assert.Equal(t, mockReqError, handlerHttpErr)
				}
			}
		})
	}
}