package update

import (
	"encoding/json"
	"errors"

	"github.com/tidwall/gjson"

	"github.com/aliyun/aliyun_assist_client/agent/log"
	"github.com/aliyun/aliyun_assist_client/agent/util"
	"github.com/aliyun/aliyun_assist_client/agent/util/osutil"
	"github.com/aliyun/aliyun_assist_client/agent/version"
)

type UpdateCheckResp struct {
	Flag         int64  `json:"flag"`
	InstanceID   string `json:"instanceId"`
	NeedUpdate   int64  `json:"need_update"`
	NextInterval int64  `json:"next_interval"`
	UpdateInfo   struct {
		FileName string `json:"file_name"`
		Md5      string `json:"md5"`
		URL      string `json:"url"`
	} `json:"update_info"`
}

type UpdateCheckReport struct {
	Os         string `json:"os"`
	Arch       string `json:"arch"`
	OsVersion  string `json:"os_version"`
	AppID      string `json:"app_id"`
	AppVersion string `json:"app_version"`
}

func FetchUpdateInfo() (*UpdateCheckResp, error) {
	report := &UpdateCheckReport{
		Os:         osutil.GetOsType(),
		AppVersion: version.AssistVersion,
		AppID:      "aliyun assistant",
		OsVersion:  osutil.GetVersion(),
		Arch:       osutil.GetOsArch(),
	}
	jsonBytes, _ := json.Marshal(*report)
	log.GetLogger().Info("UpdateCheck request: ", string(jsonBytes))

	host := util.GetUpdateService()
	responseData, err := util.HttpPost(host, string(jsonBytes), "")
	if err != nil {
		return nil, err
	}

	if !gjson.Valid(responseData) {
		return nil, errors.New("invalid json")
	}
	log.GetLogger().Info("UpdateCheck response: ", responseData)

	var resp UpdateCheckResp
	if err := json.Unmarshal([]byte(responseData), &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}
