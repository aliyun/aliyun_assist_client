package export

import (
	"bytes"
	"encoding/xml"
	"io"
	"os"
	"time"

	"github.com/aliyun/aliyun_assist_client/agent/util"
	"github.com/aliyun/aliyun_assist_client/thirdparty/sirupsen/logrus"
)

type OSSExporter struct {
	OssURL string `json:"ossUrl"`
}

type OSSExportRes struct {
	OssURL     string `json:"ossUrl"`
	StatusCode int    `json:"statusCode"`
	ErrorCode  string `json:"errorCode"`
	Msg        string `json:"msg"`
}

type OSSResp struct {
	Code    string `xml:"Code"`
	Message string `xml:"Message"`
}

var (
	OSSPutTimeout = time.Second * time.Duration(15)
)

func (o *OSSExporter) PutObjectFromContent(logger logrus.FieldLogger, content []byte) *OSSExportRes {
	reader := bytes.NewReader(content)
	return o.putObjectFromReader(logger, reader)
}

func (o *OSSExporter) PutObjectFromFile(logger logrus.FieldLogger, filePath string) *OSSExportRes {
	f, err := os.OpenFile(filePath, os.O_RDONLY, os.ModePerm)
	if err != nil {
		return o.FailedReadOutputFile(err)
	}
	defer f.Close()
	return o.putObjectFromReader(logger, f)
}

func (o *OSSExporter) putObjectFromReader(logger logrus.FieldLogger, reader io.ReadSeeker) *OSSExportRes {
	resp, err := util.HttpPUTFromReaderWithTimeout(o.OssURL, reader, OSSPutTimeout)
	if err != nil {
		if _, seekErr := reader.Seek(0, 0); seekErr != nil {
			logger.WithError(seekErr).Error("Seek to file begin failed")
			return o.FailedHTTPRequest(err)
		}
		resp, err = util.HttpPUTFromReaderWithTimeout(o.OssURL, reader, OSSPutTimeout)
	}
	if err != nil {
		return o.FailedHTTPRequest(err)
	}

	if resp.StatusCode == 200 {
		return o.Success()
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.WithError(err).Error("Read response body failed.")
		return o.RequestNonOKStatus(resp.StatusCode, resp.Status)
	}

	ossResp := &OSSResp{}
	if err := xml.Unmarshal(body, ossResp); err != nil {
		logger.WithError(err).Error("Unmarshal xml from response body failed.")
		logger.Info("Response body: ", string(body))
		return o.RequestNonOKStatus(resp.StatusCode, resp.Status)
	}

	return &OSSExportRes{
		OssURL:     o.OssURL,
		StatusCode: resp.StatusCode,
		ErrorCode:  ossResp.Code,
		Msg:        ossResp.Message,
	}
}

func (o *OSSExporter) FailedCreateOutputFile(err error) *OSSExportRes {
	return &OSSExportRes{
		OssURL:     o.OssURL,
		StatusCode: 1,
		ErrorCode:  "OutputFileCreateFailed",
		Msg:        "Create output file failed, " + err.Error(),
	}
}

func (o *OSSExporter) FailedWriteOutputFile(err error) *OSSExportRes {
	return &OSSExportRes{
		OssURL:     o.OssURL,
		StatusCode: 1,
		ErrorCode:  "OutputFileWriteFailed",
		Msg:        "An error occurred while writing output file, " + err.Error(),
	}
}

func (o *OSSExporter) FailedReadOutputFile(err error) *OSSExportRes {
	return &OSSExportRes{
		OssURL:     o.OssURL,
		StatusCode: 1,
		ErrorCode:  "OutputFileReadFailed",
		Msg:        "Read output file failed, " + err.Error(),
	}
}

func (o *OSSExporter) NoOutputFile() *OSSExportRes {
	return &OSSExportRes{
		OssURL:     o.OssURL,
		StatusCode: 1,
		ErrorCode:  "NoOutputFile",
		Msg:        "There are no output files for this task",
	}
}

func (o *OSSExporter) FailedHTTPRequest(err error) *OSSExportRes {
	return &OSSExportRes{
		OssURL:     o.OssURL,
		StatusCode: 1,
		ErrorCode:  "HTTPRequestFailed",
		Msg:        "HTTP request to oss failed, " + err.Error(),
	}
}

func (o *OSSExporter) RequestNonOKStatus(statusCode int, status string) *OSSExportRes {
	return &OSSExportRes{
		OssURL:     o.OssURL,
		StatusCode: statusCode,
		ErrorCode:  "HTTPRequestNonOKStatus",
		Msg:        "HTTP request status non OK: " + status,
	}
}

func (o *OSSExporter) Success() *OSSExportRes {
	return &OSSExportRes{
		OssURL:     o.OssURL,
		StatusCode: 200,
		ErrorCode:  "OK",
	}
}
