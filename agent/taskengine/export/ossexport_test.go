package export

import (
	"errors"
	"io"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/aliyun/aliyun_assist_client/agent/util"
	"github.com/aliyun/aliyun_assist_client/common/fileutil"
	"github.com/aliyun/aliyun_assist_client/common/requester"
	"github.com/aliyun/aliyun_assist_client/thirdparty/sirupsen/logrus"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
)

func TestPutObjectFromFile(t *testing.T) {
	mockOssUrl := "https://somebucket/somefile"
	var (
		mockHTTPStatusCode int
		mockHTTPRespBody   string
		willHTTPOK         bool
		willHTTPFail       bool
		fileContent        []byte
	)

	httpmock.Activate()
	requester.NilTransport.Set()
	defer gomonkey.ApplyFunc(util.GetHTTPTransport, func()*http.Transport {
		return nil
	}).Reset()
	defer requester.NilTransport.Clear()
	defer httpmock.DeactivateAndReset()
	httpmock.RegisterResponder("PUT", mockOssUrl,
		func(r *http.Request) (*http.Response, error) {
			content, err := io.ReadAll(r.Body)
			assert.Nil(t, err)
			assert.Equal(t, fileContent, content)

			if willHTTPFail {
				return nil, errors.New("some error")
			}
			if willHTTPOK {
				return httpmock.NewStringResponse(200, "success"), nil
			}
			return httpmock.NewStringResponse(mockHTTPStatusCode, mockHTTPRespBody), nil
		})
	fileContent = generateRandom(1024)
	sourceFile := filepath.Join(os.TempDir(), "testsource.txt")
	err := fileutil.WriteStringToFile(sourceFile, string(fileContent))
	assert.Nil(t, err)
	defer os.Remove(sourceFile)

	testCases := []struct {
		Name        string
		ExpectedRes *OSSExportRes
	}{
		{
			Name: "open_file_faield",
			ExpectedRes: &OSSExportRes{
				OssURL: mockOssUrl,
				StatusCode: 1,
				ErrorCode:  "OutputFileReadFailed",
			},
		},
		{
			Name: "http_faield",
			ExpectedRes: &OSSExportRes{
				OssURL: mockOssUrl,
				StatusCode: 1,
				ErrorCode:  "HTTPRequestFailed",
			},
		},
		{
			Name: "http_non_ok",
			ExpectedRes: &OSSExportRes{
				OssURL: mockOssUrl,
				StatusCode: 403,
				ErrorCode:  "HTTPRequestNonOKStatus",
			},
		},
		{
			Name: "no_such_bucket",
			ExpectedRes: &OSSExportRes{
				OssURL: mockOssUrl,
				StatusCode: 404,
				ErrorCode:  "NoSuchBucket",
			},
		},
		{
			Name: "ok",
			ExpectedRes: &OSSExportRes{
				OssURL: mockOssUrl,
				StatusCode: 200,
				ErrorCode:  "OK",
			},
		},
	}

	exporter := &OSSExporter{
		OssURL: mockOssUrl,
	}
	for _, tt := range testCases {
		t.Run(tt.Name, func(t *testing.T) {
			var res *OSSExportRes
			mockHTTPStatusCode = 0
			mockHTTPRespBody  = ""
			willHTTPFail = false
			willHTTPOK = true

			switch tt.Name {
			case "open_file_faield":
				res = exporter.PutObjectFromFile(logrus.New(), "/not/exist/file")
			case "http_faield":
				willHTTPFail = true
				res = exporter.PutObjectFromFile(logrus.New(), sourceFile)
			case "http_non_ok":
				willHTTPOK = false
				mockHTTPStatusCode = tt.ExpectedRes.StatusCode
				res = exporter.PutObjectFromFile(logrus.New(), sourceFile)
			case "no_such_bucket":
				willHTTPOK = false
				mockHTTPStatusCode = 404
				mockHTTPRespBody = `<?xml version="1.0" encoding="UTF-8"?>
<Error>
  <Code>NoSuchBucket</Code>
  <Message>The specified bucket does not exist.</Message>
  <RequestId>xxx</RequestId>
  <HostId>xxx.aliyuncs.com</HostId>
  <BucketName>xxx</BucketName>
  <EC>000</EC>
</Error>`
				res = exporter.PutObjectFromFile(logrus.New(), sourceFile)
			case "ok":
				res = exporter.PutObjectFromFile(logrus.New(), sourceFile)
			default:
				return
			}

			assert.Equal(t, tt.ExpectedRes.OssURL, res.OssURL)
			assert.Equal(t, tt.ExpectedRes.StatusCode, res.StatusCode)
			assert.Equal(t, tt.ExpectedRes.ErrorCode, res.ErrorCode)
		})
	}
}

func TestFailedXxx(t *testing.T) {
	exporter := &OSSExporter{
		OssURL: "https://somebucket/somefile",
	}
	err := errors.New("some error")

	res := exporter.FailedCreateOutputFile(err)
	assert.Equal(t, exporter.OssURL, res.OssURL)
	assert.Equal(t, 1, res.StatusCode)
	assert.Equal(t, "OutputFileCreateFailed", res.ErrorCode)
	assert.Equal(t, "Create output file failed, " + err.Error(), res.Msg)

	res = exporter.FailedWriteOutputFile(err)
	assert.Equal(t, exporter.OssURL, res.OssURL)
	assert.Equal(t, 1, res.StatusCode)
	assert.Equal(t, "OutputFileWriteFailed", res.ErrorCode)
	assert.Equal(t, "An error occurred while writing output file, " + err.Error(), res.Msg)

	res = exporter.NoOutputFile()
	assert.Equal(t, exporter.OssURL, res.OssURL)
	assert.Equal(t, 1, res.StatusCode)
	assert.Equal(t, "NoOutputFile", res.ErrorCode)
	assert.Equal(t, "There are no output files for this task", res.Msg)
}

func generateRandom(length int) []byte {
	if length == 0 {
		return []byte{}
	}
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return b
}
