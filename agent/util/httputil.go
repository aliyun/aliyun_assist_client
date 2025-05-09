package util

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/aliyun/aliyun_assist_client/thirdparty/sirupsen/logrus"

	"github.com/tidwall/gjson"
	"github.com/kirinlabs/HttpRequest"

	"github.com/aliyun/aliyun_assist_client/agent/log"
	"github.com/aliyun/aliyun_assist_client/common/httpbase"
	"github.com/aliyun/aliyun_assist_client/common/httputil"
	"github.com/aliyun/aliyun_assist_client/common/requester"
)

type HTTPErrHandler func(resp *HttpRequest.Response, httpErr error)

var (
	ErrHTTPCode = errors.New("http code error")

	httpPostErrHandler_ HTTPErrHandler
)

// Try not to initiate new http requests in the handler to avoid circular calls.
func SetHTTPPostErrHandler(handler HTTPErrHandler) {
	httpPostErrHandler_ = handler
}

func GetHTTPTransport() *http.Transport {
	return requester.GetHTTPTransport(log.GetLogger())
}

func HttpGet(url string) (error, string) {
	return HttpGetWithTimeout(url, 5, false)
}

func HttpGetWithTimeout(url string, timeoutSecond int, noLog bool) (error, string) {
	logger := log.GetLogger().WithFields(logrus.Fields{
		"url":     url,
		"timeout": timeoutSecond,
	})
	transport := GetHTTPTransport()
	var extraHeaders map[string]string
	var err error
	extraHeaders, _ = requester.GetExtraHTTPHeaders(logger)
	req := httputil.NewGetReq(logger, transport, timeoutSecond, extraHeaders)

	res, err := req.Get(url)
	if err != nil {
		log.GetLogger().Infoln(url, err)
		var certificateErr *tls.CertificateVerificationError
		if !errors.As(err, &certificateErr) {
			return err, ""
		}

		// tls.CertificateVerificationError encountered. Gonna re-accumulate
		// root CA certificate pool and retry requesting.
		// 1. Nil transport means working with net/http.defaultHTTPTransport
		// which does not hold the custom pool. Give up retrying
		if transport == nil {
			return err, ""
		}
		logger.Info("certificate error, reload certificates and retry")
		// 2. req.Transport recv a *http.Transport, pass a copy of
		// requester._httpTransport to it to prevent requester._httpTransport
		// being modified
		transport = transport.Clone()
		// 3. Re-accumulate root CAs and try
		requester.AccumulateRootCAs(logger)(func(certPool *x509.CertPool) bool {
			req = httputil.NewGetReq(logger, transport, timeoutSecond, extraHeaders)
			req.SetTLSClient(&tls.Config{
				RootCAs: certPool,
			})
			if res, err = req.Get(url); err == nil {
				logger.Info("certificate updated")
				requester.RefreshHTTPCas(logger, certPool)
				return false
			}

			return true
		})
		// 4. Re-accumulation ends and error still exists. Give up and raise.
		if err != nil {
			log.GetLogger().Infoln(url, err)
			return err, ""
		}
	}
	defer res.Close()
	content, _ := res.Content()
	if err == nil && res.StatusCode() > 400 {
		err = httpbase.NewStatusCodeError(res.StatusCode())
	}

	if noLog {
		// API消息体过大默认不打日志
		log.GetLogger().Debugln(url, content, err)
	} else {
		log.GetLogger().Infoln(url, content, err)
	}
	return err, content
}

func HttpPost(url string, data string, contentType string) (string, error) {
	return HttpPostWithTimeout(url, data, contentType, 5, false)
}

func HttpPostWithTimeout(url string, data string, contentType string, timeoutSecond int, noLog bool) (string, error) {
	logger := log.GetLogger().WithFields(logrus.Fields{
		"url":     url,
		"timeout": timeoutSecond,
	})
	transport := GetHTTPTransport()
	var (
		extraHeaders map[string]string
		httpReqErr error
		httpResp *HttpRequest.Response
	)
	defer func() {
		if httpReqErr != nil && httpPostErrHandler_ != nil {
			httpPostErrHandler_(httpResp, httpReqErr)
		}
	}()

	extraHeaders, _ = requester.GetExtraHTTPHeaders(logger)
	req := httputil.NewPostReq(logger, transport, contentType, timeoutSecond, extraHeaders)

	httpResp, httpReqErr = req.Post(url, data)
	if httpReqErr != nil {
		log.GetLogger().Infoln(url, httpReqErr)
		var certificateErr *tls.CertificateVerificationError
		if !errors.As(httpReqErr, &certificateErr) {
			return "", httpReqErr
		}

		// tls.CertificateVerificationError encountered. Gonna re-accumulate
		// root CA certificate pool and retry requesting.
		// 1. Nil transport means working with net/http.defaultHTTPTransport
		// which does not hold the custom pool. Give up retrying
		if transport == nil {
			return "", httpReqErr
		}
		logger.Info("certificate error, reload certificates and retry")
		// 2. req.Transport recv a *http.Transport, pass a copy of
		// requester._httpTransport to it to prevent requester._httpTransport
		// being modified
		transport = transport.Clone()
		// 3. Re-accumulate root CAs and try
		requester.AccumulateRootCAs(logger)(func(certPool *x509.CertPool) bool {
			req = httputil.NewPostReq(logger, transport, contentType, timeoutSecond, extraHeaders)
			req.SetTLSClient(&tls.Config{
				RootCAs: certPool,
			})
			if httpResp, httpReqErr = req.Post(url, data); httpReqErr == nil {
				logger.Info("certificate updated")
				requester.RefreshHTTPCas(logger, certPool)
				return false
			}

			return true
		})
		// 4. Re-accumulation ends and error still exists. Give up and raise.
		if httpReqErr != nil {
			log.GetLogger().Infoln(url, httpReqErr)
			return "", httpReqErr
		}
	}

	defer httpResp.Close()
	content, _ := httpResp.Content()

	if httpReqErr == nil && httpResp.StatusCode() > 400 {
		httpReqErr = httpbase.NewStatusCodeError(httpResp.StatusCode())
	}

	if noLog {
		// API消息体过大默认不打INFO日志
		log.GetLogger().Debugln(url, content, data, httpReqErr)
	} else {
		log.GetLogger().Infoln(url, content, data, httpReqErr)
	}

	return content, httpReqErr
}

func HttpDownlod(url string, FilePath string) error {
	client := http.Client{
		// NOTE: `transport` variable would be nil when init function fails, and
		// DefaultTransport will be used instead, thus it's safe to directly
		// reference `transport` variable.
		Transport: GetHTTPTransport(),
	}
	res, err := client.Get(url)
	if err != nil {
		return err
	}
	f, err := os.Create(FilePath)
	defer f.Close()
	if err != nil {
		return err
	}
	_, err = io.Copy(f, res.Body)
	return err
}

func HttpDownloadContext(ctx context.Context, url string, FilePath string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	client := http.Client{
		// NOTE: `transport` variable would be nil when init function fails, and
		// DefaultTransport will be used instead, thus it's safe to directly
		// reference `transport` variable.
		Transport: GetHTTPTransport(),
	}
	res, err := client.Do(req)
	if err != nil {
		return err
	}

	f, err := os.Create(FilePath)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.Copy(f, res.Body)
	return err
}

// HttpDownloadWithTimeout downloads a file from url to filePath with specified
// timeout. Check if returned error is of type *url.Error and whether url.Error.Timeout
// method returns true for timeout request.
func HttpDownloadWithTimeout(url string, filePath string, timeout time.Duration) error {
	client := http.Client{
		// NOTE: `transport` variable would be nil when init function fails, and
		// DefaultTransport will be used instead, thus it's safe to directly
		// reference `transport` variable.
		Transport: GetHTTPTransport(),
		Timeout:   timeout,
	}
	res, err := client.Get(url)
	if err != nil {
		return err
	}
	if res.StatusCode != 200 {
		return httpbase.NewStatusCodeError(res.StatusCode)
	}

	f, err := os.Create(filePath)
	defer f.Close()
	if err != nil {
		return err
	}

	_, err = io.Copy(f, res.Body)
	return err
}

func CallApi(httpMethod, url string, parameters map[string]interface{}, respObj interface{}, apiTimeoutSecond int, noLog bool) error {
	var response string
	var err error
	if httpMethod == http.MethodGet {
		// for HTTP GET, parameter map values should be string
		if len(parameters) > 0 {
			url += "?"
			var first = true
			for k, v := range parameters {
				if first {
					url += fmt.Sprintf("%s=%v", k, v)
					first = false
				} else {
					url += fmt.Sprintf("&%s=%v", k, v)
				}
			}
		}
		err, response = HttpGetWithTimeout(url, apiTimeoutSecond, noLog)
	} else {
		var data []byte
		data, err = json.Marshal(parameters)
		if err != nil {
			log.GetLogger().WithFields(logrus.Fields{
				"parameters": parameters,
			}).WithError(err).Errorln("marshal error")
			return err
		}
		response, err = HttpPostWithTimeout(url, string(data), "", apiTimeoutSecond, noLog)
	}
	if err != nil {
		log.GetLogger().WithFields(logrus.Fields{
			"url": url,
		}).WithError(err).Errorln("Failed to invoke api request")
		return err
	}
	if !gjson.Valid(response) {
		log.GetLogger().WithFields(logrus.Fields{
			"url":      url,
			"response": response,
		}).Errorln("Invalid json response")
		err = fmt.Errorf("invalid json response: %s", response)
		return err
	}
	if err := json.Unmarshal([]byte(response), respObj); err != nil {
		return err
	}
	return err
}
