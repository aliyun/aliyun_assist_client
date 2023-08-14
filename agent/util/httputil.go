package util

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strconv"
	"sync"
	"time"

	"github.com/aliyun/aliyun_assist_client/thirdparty/sirupsen/logrus"
	"github.com/google/uuid"
	"github.com/kirinlabs/HttpRequest"
	"github.com/tidwall/gjson"

	"github.com/aliyun/aliyun_assist_client/agent/log"
	"github.com/aliyun/aliyun_assist_client/agent/util/atomicutil"
	"github.com/aliyun/aliyun_assist_client/agent/util/osutil"
	"github.com/aliyun/aliyun_assist_client/agent/util/timetool"
	"github.com/aliyun/aliyun_assist_client/agent/version"
)

const (
	UserAgentHeader = "User-Agent"
)

var (
	UserAgentValue string = fmt.Sprintf("%s_%s/%s", runtime.GOOS, runtime.GOARCH, version.AssistVersion)
	CrtPath        string
	NilRequest     *atomicutil.AtomicBoolean
	CaCertPool     *x509.CertPool

	_transport                   *http.Transport
	_initializeHTTPTransportOnce sync.Once
)

type HttpErrorCode struct {
	errorCode int
}

var (
	ErrHTTPCode = errors.New("http code error")
)

func init() {
	NilRequest = &atomicutil.AtomicBoolean{}
	NilRequest.Clear()
	cpath, _ := os.Executable()
	dir, _ := filepath.Abs(filepath.Dir(cpath))
	var caCert []byte
	var err error
	CrtPath = os.Getenv("ALIYUN_ASSIST_CERT_PATH")
	if CrtPath != "" && CheckFileIsExist(CrtPath) {
		caCert, err = ioutil.ReadFile(CrtPath)
		if err != nil {
			return
		}
	} else {
		CrtPath = path.Join(dir, "config", "GlobalSignRootCA.crt")
		// log.GetLogger().Infoln("crt:", CrtPath)
		caCert, err = ioutil.ReadFile(CrtPath)
		if err != nil {
			return
		}
	}
	CaCertPool = x509.NewCertPool()
	CaCertPool.AppendCertsFromPEM(caCert)
}

func (e *HttpErrorCode) Error() string {
	return fmt.Sprintf("The error code is %d", e.errorCode)
}
func (e *HttpErrorCode) GetCode() int {
	return e.errorCode
}

func GetHTTPTransport() *http.Transport {
	if NilRequest.IsSet() {
		return nil
	}
	_initializeHTTPTransportOnce.Do(func() {
		_transport = &http.Transport{
			Proxy: GetProxyFunc(),
			DialContext: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
				DualStack: true,
			}).DialContext,
			// TLSClientConfig specifies the TLS configuration, which uses custom
			// Root CA for assist server
			TLSClientConfig: &tls.Config{
				RootCAs: CaCertPool,
			},
			// Enabled HTTP/2 protocol when `TLSClientConfig` is not nil
			ForceAttemptHTTP2:     true,
			MaxIdleConns:          100,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		}
	})

	return _transport
}

func HttpGet(url string) (error, string) {
	return HttpGetWithTimeout(url, 5, false)
}

func HttpGetWithTimeout(url string, timeout time.Duration, noLog bool) (error, string) {
	req := HttpRequest.Transport(GetHTTPTransport())

	// 设置超时时间，不设置时，默认30s
	req.SetTimeout(timeout)
	req.SetTLSClient(&tls.Config{
		RootCAs: CaCertPool,
	})

	if IsHybrid() {
		addHttpHeads(req)
	}

	// Add user-agent header
	req.SetHeaders(map[string]string{
		UserAgentHeader: UserAgentValue,
	})

	res, err := req.Get(url)
	if err != nil {
		log.GetLogger().Infoln(url, err)
		return err, ""
	}
	defer res.Close()
	content, _ := res.Content()
	if err == nil && res.StatusCode() > 400 {
		err = &HttpErrorCode{
			errorCode: res.StatusCode(),
		}
	}

	if noLog {
		// API消息体过大默认不打日志
		log.GetLogger().Debugln(url, content, err)
	} else {
		log.GetLogger().Infoln(url, content, err)
	}
	return err, content
}

func addHttpHeads(req *HttpRequest.Request) {
	u4 := uuid.New()
	str_request_id := u4.String()

	timestamp := timetool.GetAccurateTime()
	str_timestamp := strconv.FormatInt(timestamp, 10)

	var instance_id string
	var path string
	path, _ = GetHybridPath()

	content, _ := ioutil.ReadFile(path + "/instance-id")
	instance_id = string(content)

	mid, _ := GetMachineID()

	input := instance_id + mid + str_timestamp + str_request_id
	pri_key, _ := ioutil.ReadFile(path + "/pri-key")
	output := RsaSign(input, string(pri_key))
	log.GetLogger().Infoln(input, output)

	internal_ip, err := osutil.ExternalIP()
	if err == nil {
		req.SetHeaders(map[string]string{
			"X-Client-IP": internal_ip.String(),
		})
	}

	req.SetHeaders(map[string]string{
		"x-acs-instance-id": instance_id,
	})
	req.SetHeaders(map[string]string{
		"x-acs-timestamp": str_timestamp, //这也是HttpRequest包的默认设置
	})
	req.SetHeaders(map[string]string{
		"x-acs-request-id": str_request_id, //这也是HttpRequest包的默认设置
	})
	req.SetHeaders(map[string]string{
		"x-acs-signature": output, //这也是HttpRequest包的默认设置
	})
}

func HttpPost(url string, data string, contentType string) (string, error) {
	return HttpPostWithTimeout(url, data, contentType, 5, false)
}

func HttpPostWithTimeout(url string, data string, contentType string, timeout time.Duration, noLog bool) (string, error) {
	req := HttpRequest.Transport(GetHTTPTransport())
	// 设置超时时间，不设置时，默认30s
	req.SetTimeout(timeout)
	req.SetTLSClient(&tls.Config{
		RootCAs: CaCertPool,
	})

	req.SetHeaders(map[string]string{
		UserAgentHeader: UserAgentValue,
	})
	//excude Hybrid instance id
	if IsHybrid() {
		addHttpHeads(req)
	} else {
		instance_id := GetInstanceId()

		req.SetHeaders(map[string]string{
			"X-Client-Instance-ID": instance_id,
		})
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

	defer res.Close()
	content, _ := res.Content()

	if err == nil && res.StatusCode() > 400 {
		err = &HttpErrorCode{
			errorCode: res.StatusCode(),
		}
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
