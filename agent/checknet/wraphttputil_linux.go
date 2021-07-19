package checknet

import (
	"errors"

	"github.com/aliyun/aliyun_assist_client/agent/util"
)

// HttpGet calls util.HttpGet with wrapping code which issue network diagnostic
// when encoutering network error
func HttpGet(url string) (error, string) {
	err, responseContent := util.HttpGet(url)
	if err != nil && !errors.Is(err, util.ErrHTTPCode) {
		RequestNetcheck(NetcheckRequestNormal)
	} else {
		clearNeedToReport()
	}
	return err, responseContent
}

// HttpPost calls util.HttpPost with wrapping code which issue network diagnostic
// when encoutering network error
func HttpPost(url string, data string, contentType string) (string, error) {
	responseContent, err := util.HttpPost(url, data, contentType)
	if err != nil && !errors.Is(err, util.ErrHTTPCode) {
		RequestNetcheck(NetcheckRequestNormal)
	} else {
		clearNeedToReport()
	}
	return responseContent, err
}

// HttpDownload simply directly calls util.HttpDownload function without wrapping
func HttpDownlod(url string, filePath string) error {
	return util.HttpDownlod(url, filePath)
}
