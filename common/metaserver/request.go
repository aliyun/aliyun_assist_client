package metaserver

import (
	"fmt"

	"github.com/aliyun/aliyun_assist_client/thirdparty/sirupsen/logrus"
	"go.uber.org/atomic"

	"github.com/aliyun/aliyun_assist_client/common/httpbase"
	"github.com/aliyun/aliyun_assist_client/common/requester"
)

const (
	_httpTokenHeader = "X-aliyun-ecs-metadata-token"

	_acquiringTokenTTLHeader = "X-aliyun-ecs-metadata-token-ttl-seconds"
	_acquiringTokenTTLSeconds = 3600
)

var (
	_httpTokenValue atomic.String
)

func fetch(logger logrus.FieldLogger, url string, requestOptions ...httpbase.RequestOption) (string, error) {
	proxiedOptions := append([]httpbase.RequestOption{
		httpbase.WithTransport(requester.GetProxiedHTTPTransport(logger)),
	}, requestOptions...)

	content, err := fetchWithToken(url, proxiedOptions...)
	if err == nil {
		return content, nil
	}

	if statusCodeErr, ok := err.(*httpbase.StatusCodeError); ok && statusCodeErr.StatusCode() == 403 {
		logger.Warn("Failed to get metadata since token required. Trying to acquire token")
		if tokenErr := acquireToken(logger, requestOptions...); tokenErr != nil {
			return "", statusCodeErr
		}

		return fetchWithToken(url, proxiedOptions...)
	}

	return "", err
}

func fetchWithToken(url string, requestOptions ...httpbase.RequestOption) (string, error) {
	if token := _httpTokenValue.Load(); token != "" {
		requestOptions = append(requestOptions, httpbase.WithHeaders(map[string]string{
			_httpTokenHeader: token,
		}))
	}

	return httpbase.Get(url, requestOptions...)
}

func acquireToken(logger logrus.FieldLogger, requestOptions ...httpbase.RequestOption) error {
	acquiringOptions := append([]httpbase.RequestOption{
		httpbase.WithTransport(requester.GetProxiedHTTPTransport(logger)),
		httpbase.WithHeaders(map[string]string{
			_acquiringTokenTTLHeader: fmt.Sprint(_acquiringTokenTTLSeconds),
		}),
	}, requestOptions...)

	token, err := httpbase.Put("http://100.100.100.200/latest/api/token", acquiringOptions...)
	if err != nil {
		logger.WithError(err).Error("Failed to acquire token for metadata via HTTP PUT request")
		return err
	}

	logger.WithFields(logrus.Fields{
		"token": token,
	}).Info("Acquired token for metadata")
	_httpTokenValue.Store(token)
	return nil
}
