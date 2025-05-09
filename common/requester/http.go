package requester

import (
	"crypto/tls"
	"crypto/x509"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/aliyun/aliyun_assist_client/thirdparty/sirupsen/logrus"
	"github.com/aliyun/aliyun_assist_client/agent/util/atomicutil"
)

var (
	NilTransport *atomicutil.AtomicBoolean
)

var (
	_httpTransport *http.Transport
	_httpTransportLock sync.RWMutex
	_initHTTPTransportOnce sync.Once

	_proxiedHTTPTransport *http.Transport
	_initProxiedHTTPTransportOnce sync.Once
)

func init() {
	NilTransport = &atomicutil.AtomicBoolean{}
	NilTransport.Clear()
}

func GetHTTPTransport(logger logrus.FieldLogger) *http.Transport {
	if NilTransport.IsSet() {
		return nil
	}

	_initHTTPTransportOnce.Do(func() {
		_httpTransportLock.Lock()
		defer _httpTransportLock.Unlock()

		_httpTransport = unsafeGetProxiedHTTPTransport(logger)
		// TLSClientConfig specifies the TLS configuration, which uses custom
		// Root CA for assist server
		_httpTransport.TLSClientConfig = &tls.Config{
			RootCAs: GetRootCAs(logger),
		}
	})

	return _httpTransport
}

func RefreshHTTPCas(logger logrus.FieldLogger, certPool *x509.CertPool) {
	_httpTransportLock.Lock()
	defer _httpTransportLock.Unlock()
	_httpTransport.TLSClientConfig = &tls.Config{
		RootCAs: certPool,
	}
	UpdateRootCAs(logger, certPool)
}

func GetProxiedHTTPTransport(logger logrus.FieldLogger) *http.Transport {
	if NilTransport.IsSet() {
		return nil
	}

	_initProxiedHTTPTransportOnce.Do(func() {
		_proxiedHTTPTransport = unsafeGetProxiedHTTPTransport(logger)
	})

	return _proxiedHTTPTransport
}

func unsafeGetProxiedHTTPTransport(logger logrus.FieldLogger) *http.Transport {
	return &http.Transport{
		Proxy: GetProxyFunc(logger),

		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
			DualStack: true,
		}).DialContext,

		// Enabled HTTP/2 protocol when `TLSClientConfig` is not nil
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
}
