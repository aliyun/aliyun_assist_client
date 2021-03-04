package scriptmanager

import (
	"errors"

	"github.com/aliyun/aliyun_assist_client/agent/util"
)

var (
	ErrScriptFileExists = errors.New("script file existed")
)

func SaveScriptFile(savePath string, content string) error {
	if ret := util.CheckFileIsExist(savePath); ret == true {
		return ErrScriptFileExists
	}

	if err := util.WriteStringToFile(savePath, content); err != nil {
		return err
	}

	return nil
}
