package util

import (
	"net/http"
	"net/url"
	"os"
	"sync"

	"github.com/aliyun/aliyun_assist_client/thirdparty/sirupsen/logrus"
	"golang.org/x/net/http/httpproxy"

	"github.com/aliyun/aliyun_assist_client/agent/log"
)

var (
	_obtainAssistProxyEnvOnce sync.Once
	_requestAssistProxyFunc   func(*http.Request) (*url.URL, error)
)

func GetProxyFunc() func(*http.Request) (*url.URL, error) {
	_obtainAssistProxyEnvOnce.Do(func() {
		assistProxyEnv := os.Getenv("ALIYUN_ASSIST_PROXY")
		if assistProxyEnv != "" {
			log.GetLogger().WithFields(logrus.Fields{
				"ALIYUN_ASSIST_PROXY": assistProxyEnv,
			}).Infoln("Detected environment variable ALIYUN_ASSIST_PROXY for proxy setting")
		}

		proxyConfig := &httpproxy.Config{
			HTTPProxy:  assistProxyEnv,
			HTTPSProxy: assistProxyEnv,
			NoProxy:    "",
			CGI:        false,
		}
		urlProxyFunc := proxyConfig.ProxyFunc()
		_requestAssistProxyFunc = func(r *http.Request) (*url.URL, error) {
			return urlProxyFunc(r.URL)
		}
	})

	return _requestAssistProxyFunc
}
