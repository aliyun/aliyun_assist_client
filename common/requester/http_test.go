package requester

import (
	"crypto/tls"
	"crypto/x509"
	"net/http"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/aliyun/aliyun_assist_client/thirdparty/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestNilTransport(t *testing.T) {
	transport := GetHTTPTransport(logrus.New())
	assert.NotNil(t, transport)
	transport = GetHTTPTransport(logrus.New(), WithDefaultDialContextFunc, WithProxy)
	assert.NotNil(t, transport)

	NilTransport.Set()
	defer NilTransport.Clear()
	transport = GetHTTPTransport(logrus.New())
	assert.Nil(t, transport)
	transport = GetHTTPTransport(logrus.New(), WithDefaultDialContextFunc, WithProxy)
	assert.Nil(t, transport)
}

func TestRefreshHTTPCas(t *testing.T) {
	t.Run("OnNilTransport", func(t *testing.T) {
		originalTransport := _httpTransport
		_httpTransport = nil
		defer func() {
			_httpTransport = originalTransport
		}()

		expectedCertPool := x509.NewCertPool()

		patches := gomonkey.NewPatches()
		updateRootCAsCalled := false
		patches.ApplyFunc(UpdateRootCAs, func(logger logrus.FieldLogger, certPool *x509.CertPool) {
			updateRootCAsCalled = true
			assert.Truef(t, certPool == expectedCertPool, "Expected certPool to be equal to the expectedCertPool")
		})
		defer patches.Reset()

		RefreshHTTPCas(nil, expectedCertPool)

		assert.True(t, updateRootCAsCalled)
	})

	t.Run("OnExistingTransport", func(t *testing.T) {
		testTransport := &http.Transport{
			TLSClientConfig: &tls.Config{
				ServerName: "test.example.com",
			},
		}

		originalTransport := _httpTransport
		_httpTransport = testTransport
		defer func() {
			_httpTransport = originalTransport
		}()

		expectedCertPool := x509.NewCertPool()

		patches := gomonkey.NewPatches()
		updateRootCAsCalled := false
		patches.ApplyFunc(UpdateRootCAs, func(logger logrus.FieldLogger, certPool *x509.CertPool) {
			updateRootCAsCalled = true
			assert.Truef(t, certPool == expectedCertPool, "Expected certPool to be equal to the expectedCertPool")
		})
		defer patches.Reset()

		RefreshHTTPCas(nil, expectedCertPool)

		assert.True(t, updateRootCAsCalled)
		assert.NotNil(t, testTransport.TLSClientConfig)
		assert.Equal(t, expectedCertPool, testTransport.TLSClientConfig.RootCAs)
		// Validate that the TLSClientConfig object SHOULD be replaced
		assert.Empty(t, testTransport.TLSClientConfig.ServerName)
	})
}
