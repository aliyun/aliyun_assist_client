// +build windows

package kickvmhandle

import (
	"github.com/aliyun/aliyun_assist_client/agent/log"
	"github.com/aliyun/aliyun_assist_client/agent/util"
	"github.com/aliyun/aliyun_assist_client/agent/util/process"
	"strings"
	"os"
)

func stopAgant(params []string) error {
	log.GetLogger().Println("stopAgant")
	processer :=  process.ProcessCmd{}
	path, err := os.Executable()
	if err != nil {
		return err
	}
	processer.SyncRunSimple(path, strings.Split("--stop", " "), 10)
	return nil
}

func removeAgant(params []string) error {
	log.GetLogger().Println("removeAgant")
	processer :=  process.ProcessCmd{}
	path, err := os.Executable()
	if err != nil {
		return err
	}
	processer.SyncRunSimple(path, strings.Split("--remove", " "), 10)
	processer.SyncRunSimple(path, strings.Split("--stop", " "), 10)
	return nil
}

func updateAgant(params []string) error {
	log.GetLogger().Println("updateAgant")
	processer :=  process.ProcessCmd{}
	path, err := util.GetCurrentPath()
	if err != nil {
		return err
	}
	path += "aliyun_assist_update.exe"

	processer.SyncRunSimple(path, strings.Split("--check_update", " "), 10)
	return nil
}