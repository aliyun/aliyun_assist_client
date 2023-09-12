// +build linux freebsd

package kickvmhandle

import (
	"strings"

	"github.com/aliyun/aliyun_assist_client/agent/log"
	"github.com/aliyun/aliyun_assist_client/agent/util/process"
	"github.com/aliyun/aliyun_assist_client/common/pathutil"
)

func stopAgant(params []string) error {
	log.GetLogger().Println("stopAgant")
	processer :=  process.ProcessCmd{}
	processer.SyncRunSimple("aliyun-service", strings.Split("--stop", " "), 10)
	return nil
}

func removeAgant(params []string) error {
	log.GetLogger().Println("removeAgant")
	processer :=  process.ProcessCmd{}
	processer.SyncRunSimple("aliyun-service", strings.Split("--remove", " "), 10)
	processer.SyncRunSimple("aliyun-service", strings.Split("--stop", " "), 10)
	return nil
}

func updateAgant(params []string) error {
	log.GetLogger().Println("updateAgant")
	processer :=  process.ProcessCmd{}
	path, err := pathutil.GetCurrentPath()
	if err != nil {
		return err
	}
	path += "aliyun_assist_update"
	processer.SyncRunSimple(path, strings.Split("--check_update", " "), 10)
	return nil
}
