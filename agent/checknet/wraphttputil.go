// +build !linux

package checknet

import (
	"github.com/aliyun/aliyun_assist_client/agent/util"
)

// HttpGet simply directly call util.HttpGet function without wrapping
func HttpGet(url string) (error, string) {
	return util.HttpGet(url)
}

// HttpPost simply directly call util.HttpPost function without wrapping
func HttpPost(url string, data string, contentType string) (string, error) {
	return util.HttpPost(url, data, contentType)
}

// HttpDownload simply directly call util.HttpDownload function without wrapping
func HttpDownlod(url string, filePath string) error {
	return util.HttpDownlod(url, filePath)
}
