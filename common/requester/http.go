package requester

import (
	"crypto/tls"
	"crypto/x509"
	"net/http"
	"sync"
	"time"

	"github.com/aliyun/aliyun_assist_client/agent/util/atomicutil"
	"github.com/aliyun/aliyun_assist_client/thirdparty/sirupsen/logrus"
)

type HTTPTransportOption func (logger logrus.FieldLogger, transport *http.Transport)

var (
	NilTransport *atomicutil.AtomicBoolean
)

var (
	DefaultHTTPTransportOptions = []HTTPTransportOption{
		WithProxy,
		WithProvidedDialContextFunc,
		WithRootCAs,
	}

	_httpTransport *http.Transport
	_httpTransportLock sync.RWMutex
	_initHTTPTransportOnce sync.Once
)

func init() {
	NilTransport = &atomicutil.AtomicBoolean{}
	NilTransport.Clear()
}

func GetHTTPTransport(logger logrus.FieldLogger, options ...HTTPTransportOption) *http.Transport {
	if NilTransport.IsSet() {
		return nil
	}

	if len(options) > 0 {
		return newHTTPTransport(logger, options...)
	}

	_initHTTPTransportOnce.Do(func() {
		_httpTransportLock.Lock()
		defer _httpTransportLock.Unlock()

		_httpTransport = newHTTPTransport(logger, DefaultHTTPTransportOptions...)
	})

	return _httpTransport
}

func RefreshHTTPCas(logger logrus.FieldLogger, certPool *x509.CertPool) {
	_httpTransportLock.Lock()
	defer _httpTransportLock.Unlock()
	if _httpTransport != nil {
		_httpTransport.TLSClientConfig = &tls.Config{
			RootCAs: certPool,
		}
	}

	UpdateRootCAs(logger, certPool)
}

func WithDefaultDialContextFunc(logger logrus.FieldLogger, transport *http.Transport) {
	transport.DialContext = defaultDialContextFunc
}

func WithProvidedDialContextFunc(logger logrus.FieldLogger, transport *http.Transport) {
	transport.DialContext = GetDialContextFunc(logger)
}

func WithProxy(logger logrus.FieldLogger, transport *http.Transport) {
	transport.Proxy = GetProxyFunc(logger)
}

func WithRootCAs(logger logrus.FieldLogger, transport *http.Transport) {
	transport.TLSClientConfig = &tls.Config{
		RootCAs: GetRootCAs(logger),
	}
}

func newHTTPTransport(logger logrus.FieldLogger, options ...HTTPTransportOption) *http.Transport {
	transport := &http.Transport{
		// Enabled HTTP/2 protocol when `TLSClientConfig` is not nil
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	for _, option := range options {
		option(logger, transport)
	}
	return transport
}
