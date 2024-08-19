package scriptmanager

import (
	"errors"

	"github.com/aliyun/aliyun_assist_client/common/fileutil"
)

var (
	ErrScriptFileExists = errors.New("script file existed")
)

func SaveScriptFile(savePath string, content string) error {
	if ret := fileutil.CheckFileIsExist(savePath); ret == true {
		return ErrScriptFileExists
	}

	if err := fileutil.WriteStringToFile(savePath, content); err != nil {
		return err
	}

	return nil
}
