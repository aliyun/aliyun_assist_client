package pluginmanager

import (
	"github.com/aliyun/aliyun_assist_client/agent/util/process"
	"github.com/aliyun/aliyun_assist_client/agent/log"
	"github.com/aliyun/aliyun_assist_client/agent/util/osutil"
	"io"
	"syscall"
	"strings"
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
		log.GetLogger().Errorf("Get Arch: formatArch[%s] rawArch[%s]: ", formatArch, rawArch)
	}()
	formatArch = ARCH_UNKNOWN
	arch, err := osutil.GetUnameMachine()
	if err != nil {
		log.GetLogger().Errorln("Get Arch: GetUnameMachine err: ", err.Error())
	}
	arch = strings.TrimSpace(arch)
	arch = strings.ToLower(arch)
	rawArch = arch

	if strings.Contains(arch, "aarch") || strings.Contains(arch, "arm") { // arm: aarch arm
		formatArch = ARCH_ARM
	} else if strings.Contains(arch, "386") || strings.Contains(arch, "686") { // x86: i386 i686
		formatArch = ARCH_32
	} else if arch == "x86_64" { // x64: x86_64
		formatArch = ARCH_64
	} else {
		log.GetLogger().Errorln("Get Arch: unknown arch: ", arch)
		formatArch = ARCH_UNKNOWN
	}
	return
}