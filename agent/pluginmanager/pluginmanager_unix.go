//go:build linux || freebsd
// +build linux freebsd

package pluginmanager

import (
	"io"
	"strings"
	"syscall"

	"github.com/aliyun/aliyun_assist_client/agent/log"
	"github.com/aliyun/aliyun_assist_client/agent/pluginmodel"
	"github.com/aliyun/aliyun_assist_client/agent/util/osutil"
	"github.com/aliyun/aliyun_assist_client/agent/util/process"
)


func syncRunKillGroup(workingDir string, commandName string, commandArguments []string, stdoutWriter io.Writer, stderrWriter io.Writer,
	 timeOut int) (exitCode int, status int, err error) {
	processCmd := process.NewProcessCmd()
	// SyncRun 中设置了进程组id和新起的进程id一致。SyncRun返回后调用系统调用kill掉进程组
	exitCode, status, err = processCmd.SyncRun(workingDir, commandName, commandArguments, stdoutWriter, stderrWriter, nil, nil, timeOut)
	log.GetLogger().Infof("syncRunKillGroup: done, workingDir[%s] commandName[%s] commandArguments[%s] timeout[%d]", workingDir, commandName, strings.Join(commandArguments, " "), timeOut)
	if exitCode != 0 || status != process.Success || err != nil {
		log.GetLogger().Errorf("syncRunKillGroup: exitCode[%d] status[%d] err[%v], not success, will kill all child process", exitCode, status, err)
		_ = syscall.Kill(-(processCmd.Pid()), syscall.SIGKILL)
	}
	return exitCode, status, err
}

func GetArch() (formatArch string, rawArch string) {
	defer func() {
		log.GetLogger().Infof("Get Arch: formatArch[%s] rawArch[%s]: ", formatArch, rawArch)
	}()

	var err error
	rawArch, err = osutil.GetUnameMachine()
	if err != nil {
		log.GetLogger().Errorln("Get Arch: GetUnameMachine err: ", err.Error())
	}
	rawArch = strings.TrimSpace(strings.ToLower(rawArch))

	var ok bool
	formatArch, ok = pluginmodel.HostArchitecture2Model(rawArch)
	if !ok {
		log.GetLogger().Errorln("Get Arch: unknown arch: ", rawArch)
	}
	return
}
