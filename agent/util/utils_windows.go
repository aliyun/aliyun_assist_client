//go:build windows
// +build windows

package util

import (
	"bytes"
	"os/exec"
	"strings"

	"github.com/aliyun/aliyun_assist_client/common/executil"
	"github.com/aliyun/aliyun_assist_client/common/langutil"
)

func ExeCmdNoWait(cmd string) (error, int) {
	var command *exec.Cmd
	command = executil.Command("cmd", "/c", cmd)
	err := command.Start()
	if nil != err {
		return err, 0
	}
	return nil, command.Process.Pid
}

func ExeCmd(cmd string) (error, string, string) {
	var command *exec.Cmd
	command = executil.Command("cmd", "/c", cmd)
	var outInfo bytes.Buffer
	var errInfo bytes.Buffer
	command.Stdout = &outInfo
	command.Stderr = &errInfo
	err := command.Run()
	var stdout, stderr string
	if langutil.GetDefaultLang() != 0x409 {
		tmp, _ := langutil.GbkToUtf8(outInfo.Bytes())
		stdout = string(tmp)
		tmp, _ = langutil.GbkToUtf8(errInfo.Bytes())
		stderr = string(tmp)
	}
	if nil != err {
		return err, stdout, stderr
	}

	return nil, stdout, stderr
}

func IsServiceExist(ServiceName string) bool {
	var detect_str string
	detect_str = "sc query | findstr " + ServiceName
	_, stdout, _ := ExeCmd(detect_str)
	if strings.Contains(stdout, ServiceName) {
		return true
	} else {
		return false
	}
}

func IsServiceRunning(ServiceName string) bool {
	detect_str := "sc query " + ServiceName
	_, stdout, _ := ExeCmd(detect_str)
	if strings.Contains(stdout, " RUNNING") {
		return true
	}
	return false
}

func StartService(ServiceName string) error {
	if IsServiceRunning(ServiceName) {
		return nil
	}
	var err error
	err, _, _ = ExeCmd("net start " + ServiceName)
	return err
}

func StopService(ServiceName string) error {
	if !IsServiceRunning(ServiceName) {
		return nil
	}
	var err error
	err, _, _ = ExeCmd("net stop " + ServiceName)
	return err
}
