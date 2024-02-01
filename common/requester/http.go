package requester

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"net/http"
	"runtime"
	"sync"
	"time"

	"github.com/aliyun/aliyun_assist_client/thirdparty/sirupsen/logrus"

	"github.com/aliyun/aliyun_assist_client/agent/version"
)

const (
	UserAgentHeader = "User-Agent"
)

var (
	UserAgentValue string = fmt.Sprintf("%s_%s/%s", runtime.GOOS, runtime.GOARCH, version.AssistVersion)

	_httpTransport *http.Transport
	_httpTransportLock sync.RWMutex
	_initHTTPTransportOnce sync.Once
)

func GetHTTPTransport(logger logrus.FieldLogger) *http.Transport {
	_initHTTPTransportOnce.Do(func() {
		_httpTransportLock.Lock()
		defer _httpTransportLock.Unlock()
		_httpTransport = &http.Transport{
			Proxy: GetProxyFunc(logger),
			DialContext: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
				DualStack: true,
			}).DialContext,
			// TLSClientConfig specifies the TLS configuration, which uses custom
			// Root CA for assist server
			TLSClientConfig: &tls.Config{
				RootCAs: GetRootCAs(logger),
			},
			// Enabled HTTP/2 protocol when `TLSClientConfig` is not nil
			ForceAttemptHTTP2:     true,
			MaxIdleConns:          100,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
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

// PeekHTTPTransport: return a deep copy of _httpTransport. PeekHTTPTransport must be called after GetHTTPTransport
func PeekHTTPTransport(logger logrus.FieldLogger) *http.Transport {
	_httpTransportLock.RLock()
	defer _httpTransportLock.RUnlock()
	clonedHttpTransport := _httpTransport.Clone()
	return clonedHttpTransport
}

type HttpErrorCode struct {
	errorCode int
}

func NewHttpErrorCode(code int) *HttpErrorCode {
	return &HttpErrorCode{
		errorCode: code,
	}
}

func (e *HttpErrorCode) Error() string {
	return fmt.Sprintf("The error code is %d", e.errorCode)
}
func (e *HttpErrorCode) GetCode() int {
	return e.errorCode
}
