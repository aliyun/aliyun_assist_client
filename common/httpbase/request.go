package httpbase

import (
	"crypto/tls"
	"net/http"
	"time"

	"github.com/kirinlabs/HttpRequest"
)

type RequestOption func (request *HttpRequest.Request)

func WithHeaders(headers map[string]string) RequestOption {
	return func(request *HttpRequest.Request) {
		request.SetHeaders(headers)
	}
}

func WithTimeoutInSeconds(timeoutInSeconds int) RequestOption {
	return func(request *HttpRequest.Request) {
		// IMPORTANT NOTE: Although time.Duration type is used for the argument
		// of (*HttpRequest.Request).SetTimeout(d time.Duration), the actual
		// unit is not nanosecond but second, since the value would be
		// internally multiplied by base.
		//
		// See link below for implementation detail:
		// https://github.com/kirinlabs/HttpRequest/blob/432628e833bda77cc426fc1bee9825a13f6b4df1/request.go#L105
		request.SetTimeout(time.Duration(timeoutInSeconds))
	}
}

func WithTransport(transport *http.Transport) RequestOption {
	return func(request *HttpRequest.Request) {
		request.Transport(transport)
	}
}

func WithTLSClientConfig(tlsClientConfig *tls.Config) RequestOption {
	return func(request *HttpRequest.Request) {
		request.SetTLSClient(tlsClientConfig)
	}
}

func buildRequest(options ...RequestOption) *HttpRequest.Request {
	request := HttpRequest.NewRequest()
	for _, option := range options {
		option(request)
	}
	return request
}
