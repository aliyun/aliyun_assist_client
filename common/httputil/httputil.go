package httputil

import (
	"net/http"
	"time"

	// _ "github.com/aliyun/aliyun_assist_client/common/apiserver"
	"github.com/aliyun/aliyun_assist_client/common/httpbase"
	"github.com/aliyun/aliyun_assist_client/thirdparty/sirupsen/logrus"
	"github.com/kirinlabs/HttpRequest"
)

func NewPostReq(logger logrus.FieldLogger, transport *http.Transport, contentType string, timeoutSecond int, headers map[string]string) *HttpRequest.Request {
	req := HttpRequest.Transport(transport)
	// IMPORTANT NOTE: Although time.Duration type is used for the argument of
	// (*HttpRequest.Request).SetTimeout(d time.Duration), the actual unit is
	// not nanosecond but second, since the value would be internally multiplied
	// by base.
	//
	// See link below for implementation detail:
	// https://github.com/kirinlabs/HttpRequest/blob/432628e833bda77cc426fc1bee9825a13f6b4df1/request.go#L105
	req.SetTimeout(time.Duration(timeoutSecond))

	req.SetHeaders(map[string]string{
		httpbase.UserAgentHeader: httpbase.UserAgentValue,
	})
	if headers != nil {
		req.SetHeaders(headers)
	}

	// 设置Headers
	if contentType == "text" {
		req.SetHeaders(map[string]string{
			"Content-Type": "text/plain; charset=utf-8", //这也是HttpRequest包的默认设置
		})
	} else {
		req.SetHeaders(map[string]string{
			"Content-Type": "application/json; charset=utf-8", //这也是HttpRequest包的默认设置
		})
	}
	return req
}

func NewGetReq(logger logrus.FieldLogger, transport *http.Transport, timeoutSecond int, headers map[string]string) *HttpRequest.Request {
	req := HttpRequest.Transport(transport)
	// IMPORTANT NOTE: Although time.Duration type is used for the argument of
	// (*HttpRequest.Request).SetTimeout(d time.Duration), the actual unit is
	// not nanosecond but second, since the value would be internally multiplied
	// by base.
	//
	// See link below for implementation detail:
	// https://github.com/kirinlabs/HttpRequest/blob/432628e833bda77cc426fc1bee9825a13f6b4df1/request.go#L105
	req.SetTimeout(time.Duration(timeoutSecond))

	req.SetHeaders(map[string]string{
		httpbase.UserAgentHeader: httpbase.UserAgentValue,
	})
	if headers != nil {
		req.SetHeaders(headers)
	}
	return req
}
