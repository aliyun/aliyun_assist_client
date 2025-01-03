package util

import (
	"config_ecs_instance_connect/agent/log"
	"errors"
	"github.com/kirinlabs/HttpRequest"
	"net"
	"net/http"
	"time"
)

func GetHTTPTransport() *http.Transport {
	_transport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   10 * time.Second,
			KeepAlive: 10 * time.Second,
		}).DialContext,
	}

	return _transport
}

func HttpGet(url string) (string, error) {
	return HttpGetWithHeaders(url, nil)
}

func HttpGetWithHeaders(url string, headers map[string]string) (string, error) {
	req := HttpRequest.Transport(GetHTTPTransport())
	// 设置超时时间，不设置时，默认30s
	req.SetTimeout(time.Second * 10)
	req.SetHeaders(headers)

	res, err := req.Get(url)
	if err != nil {
		log.Info(url, err)
		return "", err
	}

	defer res.Close()
	content, _ := res.Content()
	if err == nil && res.StatusCode() > 400 {
		err = errors.New("http code error" + string(res.StatusCode()))
	}

	log.Info(url, content, err)

	return content, err
}

func HttpPutWithHeaders(url string, headers map[string]string) (string, error) {
	req := HttpRequest.Transport(GetHTTPTransport())
	req.SetTimeout(time.Second * 10)
	req.SetHeaders(headers)

	res, err := req.Put(url)
	if err != nil {
		log.Info(url, err)
		return "", err
	}
	defer res.Close()
	content, _ := res.Content()
	if err == nil && res.StatusCode() > 400 {
		err = errors.New("http code error" + string(res.StatusCode()))
	}

	log.Info(url, content, err)

	return content, err
}

func HttpPost(url string, data string, contentType string) (string, error) {
	req := HttpRequest.Transport(GetHTTPTransport())
	// 设置超时时间，不设置时，默认30s
	req.SetTimeout(time.Second * 10)

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
		err = errors.New("http code error: " + string(res.StatusCode()))
	}

	log.Info(url, content, err)

	return content, err

}
