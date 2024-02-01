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
	"github.com/kirinlabs/HttpRequest"
	"github.com/tidwall/gjson"

	"github.com/aliyun/aliyun_assist_client/agent/log"
	"github.com/aliyun/aliyun_assist_client/agent/util/atomicutil"
	_ "github.com/aliyun/aliyun_assist_client/common/apiserver"
	"github.com/aliyun/aliyun_assist_client/common/requester"
)

var (
	NilRequest     *atomicutil.AtomicBoolean
)

var (
	ErrHTTPCode = errors.New("http code error")
)

func init() {
	NilRequest = &atomicutil.AtomicBoolean{}
	NilRequest.Clear()
}

func GetHTTPTransport() *http.Transport {
	if NilRequest.IsSet() {
		return nil
	}

	return requester.GetHTTPTransport(log.GetLogger())
}

func HttpGet(url string) (error, string) {
	return HttpGetWithTimeout(url, 5, false)
}

func HttpGetWithTimeout(url string, timeout time.Duration, noLog bool) (error, string) {
	req := HttpRequest.Transport(GetHTTPTransport())
	logger := log.GetLogger().WithFields(logrus.Fields{
		"url": url,
		"timeout": timeout.Seconds(),
	})
	// 设置超时时间，不设置时，默认30s
	req.SetTimeout(timeout)

	// Add user-agent header
	req.SetHeaders(map[string]string{
		requester.UserAgentHeader: requester.UserAgentValue,
	})
	if extraHeaders, err := requester.GetExtraHTTPHeaders(log.GetLogger()); extraHeaders != nil {
		req.SetHeaders(extraHeaders)
	} else if err != nil {
		log.GetLogger().WithError(err).Error("Failed to construct extra HTTP headers")
	}

	res, err := req.Get(url)
	if err != nil {
		log.GetLogger().Infoln(url, err)
		if errors.Is(err, x509.UnknownAuthorityError{}) {
			logger.Info("certificate error, reload certificates and retry")
			// req.Transport recv a *http.Transport, pass a copy of requester._httpTransport to it to prevent 
			// requester._httpTransport being modified
			req.Transport(requester.PeekHTTPTransport(logger))
			certPool := requester.PeekRefreshedRootCAs(logger)
			req.SetTLSClient(&tls.Config{
				RootCAs: certPool,
			})
			if res, err = req.Get(url); err == nil {
				logger.Info("certificate updated")
				requester.RefreshHTTPCas(logger, certPool)
			} else {
				log.GetLogger().Infoln(url, err)
				return err, ""
			}
		} else {
			return err, ""
		}
	}
	defer res.Close()
	content, _ := res.Content()
	if err == nil && res.StatusCode() > 400 {
		err = requester.NewHttpErrorCode(res.StatusCode())
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

func HttpPostWithTimeout(url string, data string, contentType string, timeout time.Duration, noLog bool) (string, error) {
	req := HttpRequest.Transport(GetHTTPTransport())
	logger := log.GetLogger().WithFields(logrus.Fields{
		"url": url,
		"timeout": timeout.Seconds(),
	})
	// 设置超时时间，不设置时，默认30s
	req.SetTimeout(timeout)

	req.SetHeaders(map[string]string{
		requester.UserAgentHeader: requester.UserAgentValue,
	})
	if extraHeaders, err := requester.GetExtraHTTPHeaders(log.GetLogger()); extraHeaders != nil {
		req.SetHeaders(extraHeaders)
	} else if err != nil {
		log.GetLogger().WithError(err).Error("Failed to construct extra HTTP headers")
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

	res, err := req.Post(url, data)

	if err != nil {
		log.GetLogger().Infoln(url, err)
		if errors.Is(err, x509.UnknownAuthorityError{}) {
			logger.Info("certificate error, reload certificates and retry")
			// req.Transport recv a *http.Transport, pass a copy of requester._httpTransport to it to prevent 
			// requester._httpTransport being modified
			req.Transport(requester.PeekHTTPTransport(logger))
			certPool := requester.PeekRefreshedRootCAs(logger)
			req.SetTLSClient(&tls.Config{
				RootCAs: certPool,
			})
			if res, err = req.Get(url); err == nil {
				logger.Info("certificate updated")
				requester.RefreshHTTPCas(logger, certPool)
			} else {
				log.GetLogger().Infoln(url, err)
				return "", err
			}
		} else {
			return "", err
		}
	}

	defer res.Close()
	content, _ := res.Content()

	if err == nil && res.StatusCode() > 400 {
		err = requester.NewHttpErrorCode(res.StatusCode())
	}

	if noLog {
		// API消息体过大默认不打INFO日志
		log.GetLogger().Debugln(url, content, data, err)
	} else {
		log.GetLogger().Infoln(url, content, data, err)
	}
	return content, err

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
		return requester.NewHttpErrorCode(res.StatusCode)
	}

	f, err := os.Create(filePath)
	defer f.Close()
	if err != nil {
		return err
	}

	_, err = io.Copy(f, res.Body)
	return err
}

func CallApi(httpMethod, url string, parameters map[string]interface{}, respObj interface{}, apiTimeout time.Duration, noLog bool) error {
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
		err, response = HttpGetWithTimeout(url, apiTimeout, noLog)
	} else {
		data, err := json.Marshal(parameters)
		if err != nil {
			log.GetLogger().WithFields(logrus.Fields{
				"parameters": parameters,
			}).WithError(err).Errorln("marshal error")
			return err
		}
		response, err = HttpPostWithTimeout(url, string(data), "", apiTimeout, noLog)
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
		if err == nil {
			err = fmt.Errorf("invalid json response: %s", response)
		}
		return err
	}
	if err := json.Unmarshal([]byte(response), respObj); err != nil {
		return err
	}
	return err
}
