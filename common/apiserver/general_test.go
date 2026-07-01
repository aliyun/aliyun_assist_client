package apiserver

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"net"
	"net/http"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/aliyun/aliyun_assist_client/thirdparty/sirupsen/logrus"
	"github.com/kirinlabs/HttpRequest"
	"github.com/stretchr/testify/assert"

	"github.com/aliyun/aliyun_assist_client/common/httpbase"
	"github.com/aliyun/aliyun_assist_client/common/httputil"
	"github.com/aliyun/aliyun_assist_client/common/requester"
)

func TestHttpGetWithSpecifiedHeader(t *testing.T) {
	logrus.SetLevel(logrus.WarnLevel)
	logger := logrus.StandardLogger()

	const expectedUrl = "https://example.com/test"
	expectedHeaders := map[string]string{
		"Authorization": "Bearer token123",
		"Content-Type":  "application/json",
	}
	guardTransport := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			panic("GUARD: DialContext method of mock transport SHOULD NEVER be called")
		},
	}

	t.Run("SuccessfulRequest", func(t *testing.T) {
		patches := gomonkey.ApplyFunc(requester.GetHTTPTransport, func(logger logrus.FieldLogger) *http.Transport {
			return guardTransport
		})

		// Mock httpGetWithTransport
		expectedContent := "success response"
		patches.ApplyFunc(httpGetWithTransport, func(logger logrus.FieldLogger, url string, headers map[string]string, transport *http.Transport) (string, error) {
			assert.Equal(t, expectedUrl, url)
			assert.Equal(t, expectedHeaders, headers)
			assert.Equal(t, guardTransport, transport)
			return expectedContent, nil
		})
		defer patches.Reset()

		content, err := HttpGetWithSpecifiedHeader(logger, expectedUrl, expectedHeaders)
		assert.NoError(t, err)
		assert.Equal(t, expectedContent, content)
	})

	t.Run("RequestWithError", func(t *testing.T) {
		patches := gomonkey.ApplyFunc(requester.GetHTTPTransport, func(logger logrus.FieldLogger, options ...requester.HTTPTransportOption) *http.Transport {
			return guardTransport
		})

		// Mock httpGetWithTransport with error
		expectedError := errors.New("network error")
		patches.ApplyFunc(httpGetWithTransport, func(logger logrus.FieldLogger, url string, headers map[string]string, transport *http.Transport) (string, error) {
			assert.Equal(t, expectedUrl, url)
			assert.Equal(t, expectedHeaders, headers)
			assert.Equal(t, guardTransport, transport)
			return "", expectedError
		})
		defer patches.Reset()

		content, err := HttpGetWithSpecifiedHeader(logger, expectedUrl, expectedHeaders)
		assert.Error(t, err)
		assert.Equal(t, expectedError, err)
		assert.Equal(t, "", content)
	})

	t.Run("NilHeaders", func(t *testing.T) {
		patches := gomonkey.ApplyFunc(requester.GetHTTPTransport, func(logger logrus.FieldLogger, options ...requester.HTTPTransportOption) *http.Transport {
			return guardTransport
		})

		// Mock httpGetWithTransport
		expectedContent := "response body"
		patches.ApplyFunc(httpGetWithTransport, func(logger logrus.FieldLogger, url string, headers map[string]string, transport *http.Transport) (string, error) {
			assert.Equal(t, expectedUrl, url)
			assert.Nil(t, headers)
			assert.Equal(t, guardTransport, transport)
			return expectedContent, nil
		})
		defer patches.Reset()

		content, err := HttpGetWithSpecifiedHeader(logger, expectedUrl, nil)
		assert.NoError(t, err)
		assert.Equal(t, expectedContent, content)
	})

	t.Run("NilTransport", func(t *testing.T) {
		patches := gomonkey.ApplyFunc(requester.GetHTTPTransport, func(logger logrus.FieldLogger, options ...requester.HTTPTransportOption) *http.Transport {
			return nil
		})

		// Mock httpGetWithTransport
		expectedContent := "response body"
		patches.ApplyFunc(httpGetWithTransport, func(logger logrus.FieldLogger, url string, headers map[string]string, transport *http.Transport) (string, error) {
			assert.Equal(t, expectedUrl, url)
			assert.Equal(t, expectedHeaders, headers)
			assert.Nil(t, transport)
			return expectedContent, nil
		})
		defer patches.Reset()

		content, err := HttpGetWithSpecifiedHeader(logger, expectedUrl, expectedHeaders)
		assert.NoError(t, err)
		assert.Equal(t, expectedContent, content)
	})
}

func TestHttpGetWithTransport(t *testing.T) {
	logrus.SetLevel(logrus.WarnLevel)
	logger := logrus.StandardLogger()

	const expectedUrl = "https://example.com/url/for/test"
	expectedHeaders := map[string]string{
		"X-Prefix-Has-Been-Deprecated": "See RFC 6648",
	}
	guardTransport := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			panic("GUARD: DialContext method of mock transport SHOULD NEVER be called")
		},
	}

	t.Run("NormalCase", func(t *testing.T) {
		// Mock httputil.NewGetReq function to mock request
		mockRequest := &HttpRequest.Request{}
		patches := gomonkey.ApplyFunc(httputil.NewGetReq, func(logger logrus.FieldLogger, transport *http.Transport, timeoutSeconds int, header map[string]string) *HttpRequest.Request {
			assert.Equal(t, guardTransport, transport)
			assert.Equal(t, expectedHeaders, header)
			return mockRequest
		})

		// Mock (*HttpRequest.Request).Get method to mock response
		mockResponse := &HttpRequest.Response{}
		patches.ApplyMethodFunc(mockRequest, "Get", func (url string, data ...interface{}) (*HttpRequest.Response, error) {
			assert.Equal(t, expectedUrl, url)
			return mockResponse, nil
		})

		// Mock HttpRequest.Response methods
		patches.ApplyMethodFunc(mockResponse, "Close", func() error {
			return nil
		})
		patches.ApplyMethodFunc(mockResponse, "Content", func() (string, error) {
			return "ok", nil
		})
		patches.ApplyMethodFunc(mockResponse, "StatusCode", func() int {
			return http.StatusOK
		})
		defer patches.Reset()

		content, err := httpGetWithTransport(logger, expectedUrl, expectedHeaders, guardTransport)
		assert.NoError(t, err)
		assert.Equal(t, "ok", content)
	})

	t.Run("HTTPGetErrorWithoutResponse", func(t *testing.T) {
		expectedErr := errors.New("mock error")

		// Mock httputil.NewGetReq
		mockRequest := &HttpRequest.Request{}
		patches := gomonkey.ApplyFunc(httputil.NewGetReq, func(logger logrus.FieldLogger, transport *http.Transport, timeoutSeconds int, header map[string]string) *HttpRequest.Request {
			assert.Equal(t, guardTransport, transport)
			assert.Equal(t, expectedHeaders, header)
			return mockRequest
		})

		// Mock (*HttpRequest.Request).Get method to return error
		patches.ApplyMethodFunc(mockRequest, "Get", func(url string, data ...interface{}) (*HttpRequest.Response, error) {
			assert.Equal(t, expectedUrl, url)
			return nil, expectedErr
		})
		defer patches.Reset()

		content, err := httpGetWithTransport(logger, expectedUrl, expectedHeaders, guardTransport)
		assert.ErrorIs(t, err, expectedErr)
		assert.Equal(t, "", content)
	})

	t.Run("CertificateVerificationErrorThenRecover", func(t *testing.T) {
		// Mock httputil.NewGetReq
		mockRequest := &HttpRequest.Request{}
		newGetReqCount := 0
		patches := gomonkey.ApplyFunc(httputil.NewGetReq, func(logger logrus.FieldLogger, transport *http.Transport, timeoutSeconds int, header map[string]string) *HttpRequest.Request {
			newGetReqCount++
			if newGetReqCount == 1 {
				assert.Equal(t, guardTransport, transport)
			} else {
				assert.NotEqual(t, guardTransport, transport)
			}

			assert.Equal(t, expectedHeaders, header)
			return mockRequest
		})

		// Mock (*HttpRequest.Request).Get method
		getRequestCount := 0
		mockResponse := &HttpRequest.Response{}
		patches.ApplyMethodFunc(mockRequest, "Get", func(url string, data ...interface{}) (*HttpRequest.Response, error) {
			assert.Equal(t, expectedUrl, url)

			getRequestCount++
			if getRequestCount == 1 {
				return nil, &tls.CertificateVerificationError{}
			} else if getRequestCount == 2 {
				return mockResponse, nil
			}
			return nil, &tls.CertificateVerificationError{}
		})

		// Mock SetTLSClient
		patches.ApplyMethod(mockRequest, "SetTLSClient", func (r *HttpRequest.Request, config *tls.Config) *HttpRequest.Request {
			// Do nothing
			return r
		})

		// Mock HttpRequest.Response methods
		patches.ApplyMethodFunc(mockResponse, "Close", func() error {
			return nil
		})
		patches.ApplyMethodFunc(mockResponse, "Content", func() (string, error) {
			return "ok", nil
		})
		patches.ApplyMethodFunc(mockResponse, "StatusCode", func() int {
			return http.StatusOK
		})

		// Mock requester.AccumulateRootCAs
		accumulateCalled := false
		patches.ApplyFunc(requester.AccumulateRootCAs, func(logger logrus.FieldLogger) func(func(*x509.CertPool) bool) {
			return func(f func(*x509.CertPool) bool) {
				certPool := x509.NewCertPool()
				accumulateCalled = true
				f(certPool)
			}
		})

		// Mock requester.RefreshHTTPCas
		refreshCalled := false
		patches.ApplyFunc(requester.RefreshHTTPCas, func(logger logrus.FieldLogger, cas *x509.CertPool) {
			refreshCalled = true
		})

		// (*http.Transport).Clone() does not need to be mocked
		defer patches.Reset()

		content, err := httpGetWithTransport(logger, expectedUrl, expectedHeaders, guardTransport)
		assert.NoError(t, err)
		assert.Equal(t, "ok", content)
		assert.True(t, accumulateCalled)
		assert.True(t, refreshCalled)
	})

	t.Run("CertificateVerificationErrorNoRecovery", func(t *testing.T) {
		expectedErr := &tls.CertificateVerificationError{}

		// Mock httputil.NewGetReq
		mockRequest := &HttpRequest.Request{}
		newGetReqCount := 0
		patches := gomonkey.ApplyFunc(httputil.NewGetReq, func(logger logrus.FieldLogger, transport *http.Transport, timeoutSeconds int, header map[string]string) *HttpRequest.Request {
			newGetReqCount++
			if newGetReqCount == 1 {
				assert.Equal(t, guardTransport, transport)
			} else {
				assert.NotEqual(t, guardTransport, transport)
			}
			assert.Equal(t, expectedHeaders, header)
			return mockRequest
		})

		// Mock (*HttpRequest.Request).Get method to return certificate verification error
		patches.ApplyMethodFunc(mockRequest, "Get", func(url string, data ...interface{}) (*HttpRequest.Response, error) {
			assert.Equal(t, expectedUrl, url)
			return nil, expectedErr
		})

		// Mock requester.AccumulateRootCAs
		patches.ApplyFunc(requester.AccumulateRootCAs, func(logger logrus.FieldLogger) func(func(*x509.CertPool) bool) {
			return func(f func(*x509.CertPool) bool) {
				certPool := x509.NewCertPool()
				f(certPool) // Always fails
			}
		})

		defer patches.Reset()

		content, err := httpGetWithTransport(logger, expectedUrl, expectedHeaders, guardTransport)
		assert.ErrorIs(t, err, expectedErr)
		assert.Equal(t, "", content)
	})

	t.Run("NilTransportCertificateVerificationError", func(t *testing.T) {
		expectedErr := &tls.CertificateVerificationError{}

		// Mock httputil.NewGetReq
		mockRequest := &HttpRequest.Request{}
		patches := gomonkey.ApplyFunc(httputil.NewGetReq, func(logger logrus.FieldLogger, transport *http.Transport, timeoutSeconds int, header map[string]string) *HttpRequest.Request {
			assert.Nil(t, transport)
			assert.Equal(t, expectedHeaders, header)
			return mockRequest
		})

		// Mock (*HttpRequest.Request).Get method to return certificate verification error
		patches.ApplyMethodFunc(mockRequest, "Get", func(url string, data ...interface{}) (*HttpRequest.Response, error) {
			assert.Equal(t, expectedUrl, url)
			return nil, expectedErr
		})

		defer patches.Reset()

		content, err := httpGetWithTransport(logger, expectedUrl, expectedHeaders, nil)
		assert.ErrorIs(t, err, expectedErr)
		assert.Equal(t, "", content)
	})

	t.Run("RequestReturnsStatus500", func(t *testing.T) {
		// Mock httputil.NewGetReq
		mockRequest := &HttpRequest.Request{}
		patches := gomonkey.ApplyFunc(httputil.NewGetReq, func(logger logrus.FieldLogger, transport *http.Transport, timeoutSeconds int, header map[string]string) *HttpRequest.Request {
			assert.Equal(t, guardTransport, transport)
			assert.Equal(t, expectedHeaders, header)
			return mockRequest
		})

		// Mock (*HttpRequest.Request).Get method
		mockResponse := &HttpRequest.Response{}
		patches.ApplyMethodFunc(mockRequest, "Get", func(url string, data ...interface{}) (*HttpRequest.Response, error) {
			assert.Equal(t, expectedUrl, url)
			return mockResponse, nil
		})

		// Mock HttpRequest.Response methods
		patches.ApplyMethodFunc(mockResponse, "Close", func() error {
			return nil
		})
		patches.ApplyMethodFunc(mockResponse, "Content", func() (string, error) {
			return "error content", nil
		})
		patches.ApplyMethodFunc(mockResponse, "StatusCode", func() int {
			return 500
		})
		defer patches.Reset()

		content, err := httpGetWithTransport(logger, expectedUrl, expectedHeaders, guardTransport)
		assert.Error(t, err)
		assert.IsType(t, &httpbase.StatusCodeError{}, err)
		assert.Equal(t, "error content", content) // Content still returned
	})
}

func TestCacheRegionId(t *testing.T) {
	for _, tc := range []struct {
		name             string
		expectedRegionId string
	}{
		{"EmptyRegionId", ""},
		{"NonEmptyRegionId", "cn-test"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			patches := gomonkey.NewPatches()
			setRegionIdCalled := false
			patches.ApplyFunc(requester.SetRegionId, func(regionId string) {
				setRegionIdCalled = true
				assert.Equal(t, tc.expectedRegionId, regionId)
			})
			saveRegionIdCalled := false
			patches.ApplyMethodFunc(regionidFileProvider, "SaveRegionId", func(_ logrus.FieldLogger, regionId string) {
				saveRegionIdCalled = true
				assert.Equal(t, tc.expectedRegionId, regionId)
			})
			defer patches.Reset()

			cacheRegionId(nil, tc.expectedRegionId)
			assert.True(t, setRegionIdCalled)
			assert.True(t, saveRegionIdCalled)
		})
	}
}
