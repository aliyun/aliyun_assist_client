package util

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/kirinlabs/HttpRequest"

	"github.com/aliyun/aliyun_assist_client/agent/log"
	"github.com/aliyun/aliyun_assist_client/agent/util/osutil"
	"github.com/aliyun/aliyun_assist_client/agent/util/timetool"
	"github.com/aliyun/aliyun_assist_client/agent/version"
)

const (
	UserAgentHeader = "User-Agent"
)

var (
	UserAgentValue string
	CrtPath        string
	CaCertPool     *x509.CertPool
)
var transport *http.Transport

func init() {
	cpath, _ := os.Executable()
	dir, _ := filepath.Abs(filepath.Dir(cpath))
	CrtPath = path.Join(dir, "config", "GlobalSignRootCA.crt")
	//log.GetLogger().Infoln("crt:", CrtPath)
	caCert, err := ioutil.ReadFile(CrtPath)
	if err != nil {
		return
	}
	CaCertPool = x509.NewCertPool()
	CaCertPool.AppendCertsFromPEM(caCert)

	transport = &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
			DualStack: true,
		}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
}
func InitUserAgentValue() {
	UserAgentValue = fmt.Sprintf("%s_%s/%s", osutil.GetOsType(), osutil.GetOsArch(), version.AssistVersion)
}

func HttpGet(url string) (error, string) {
	req := HttpRequest.Transport(transport)
	// 设置超时时间，不设置时，默认30s
	req.SetTimeout(5)
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
		err = errors.New("http code error")
	}

	log.GetLogger().Infoln(url, content, err)
	return err, content
}

func addHttpHeads(req *HttpRequest.Request) {
	u4 := uuid.New()
	str_request_id := u4.String()

	timestamp := timetool.GetAccurateTime()
	str_timestamp := strconv.FormatInt(timestamp, 10)

	var instance_id string
	path, _ := GetHybridPath()

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
	req := HttpRequest.Transport(transport)
	// 设置超时时间，不设置时，默认30s
	req.SetTimeout(5)
	req.SetTLSClient(&tls.Config{
		RootCAs: CaCertPool,
	})
	if IsHybrid() {
		addHttpHeads(req)
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

	//instance_id := GetInstanceId();

	req.SetHeaders(map[string]string{
		UserAgentHeader:        UserAgentValue,
		"X-Client-Instance-ID": "123",
	})

	res, err := req.Post(url, data)

	defer res.Close()
	content, _ := res.Content()

	if err == nil && res.StatusCode() > 400 {
		err = errors.New("http code error")
	}

	log.GetLogger().Infoln(url, content, data, err)
	return content, err

}

func HttpDownlod(url string, FilePath string) error {
	res, err := http.Get(url)
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

// HttpDownloadWithTimeout downloads a file from url to filePath with specified
// timeout. Check if returned error is of type *url.Error and whether url.Error.Timeout
// method returns true for timeout request.
func HttpDownloadWithTimeout(url string, filePath string, timeout time.Duration) error {
	client := http.Client{
		Timeout: timeout,
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
