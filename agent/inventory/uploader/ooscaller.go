package uploader

import (
	"bytes"
	"compress/flate"
	"encoding/base64"
	"fmt"
	"net/http"

	"github.com/aliyun/aliyun_assist_client/agent/inventory/model"
	"github.com/aliyun/aliyun_assist_client/agent/log"
	"github.com/aliyun/aliyun_assist_client/agent/util"
	"github.com/aliyun/aliyun_assist_client/agent/util/jsonutil"
)

type OOSCaller interface {
	PutInventory(input *model.PutInventoryInput) (err error)
}

type OOSCallerImpl struct {
}

type OOSResult struct {
	RequestId string
}

type ApiResponse struct {
	ErrCode string     `json:"errCode"`
	ErrMsg  string     `json:"errMsg"`
	Result  *OOSResult `json:"result"`
}

func PutInventory(items string, resp *ApiResponse) error {
	url := util.GetPutInventoryService()
	parameters := map[string]interface{}{"items": items}
	err := util.CallApi(http.MethodPost, url, parameters, resp, 10, true)
	if err == nil {
		if resp.ErrCode >= "400" {
			err = fmt.Errorf("%s %s", resp.ErrCode, resp.ErrMsg)
			if resp.Result != nil {
				log.GetLogger().Errorf("PutInventory failed, %s, %s, requestId %s", resp.ErrCode, resp.ErrMsg, resp.Result.RequestId)
			}
		} else if resp.Result != nil {
			log.GetLogger().Infof("PutInventory success, requestId %s", resp.Result.RequestId)
		}
	}
	return err
}

func Compress(input string) (output string) {
	buf := new(bytes.Buffer)
	w, _ := flate.NewWriter(buf, 7)
	w.Write([]byte(input))
	w.Flush()

	output = base64.StdEncoding.EncodeToString(buf.Bytes())
	return
}

func (i *OOSCallerImpl) PutInventory(input *model.PutInventoryInput) (err error) {
	itemString, err := jsonutil.Marshal(input.Items)
	if err != nil {
		return err
	}
	compressString := Compress(itemString)
	var resp *ApiResponse = &ApiResponse{}
	err = PutInventory(compressString, resp)
	return err
}

func NewOOSCallerImpl() (*OOSCallerImpl, error) {
	var ooscaller = OOSCallerImpl{}
	return &ooscaller, nil
}
