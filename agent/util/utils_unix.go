// +build linux freebsd

package util

import (
	"os/exec"
	"syscall"
	"bytes"
	"strings"
	"errors"
)

func ExeCmdNoWait(cmd string) (error, int) {
	var command *exec.Cmd
	command = exec.Command("sh", "-c", cmd)
	command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	err := command.Start()
	if nil != err {
		return err, 0
	}
	return nil, command.Process.Pid
}


func ExeCmd(cmd string) (error, string, string) {
	var command *exec.Cmd
	command = exec.Command("sh", "-c", cmd)
	var outInfo bytes.Buffer
	var errInfo bytes.Buffer
	command.Stdout = &outInfo
	command.Stderr = &errInfo
	err := command.Run()
	if nil != err {
		return err, outInfo.String(), errInfo.String()
	}

	return nil, outInfo.String(), errInfo.String()
}

func IsSystemdLinux() bool {
	detect_str := "[[ `systemctl` =~ -.mount ]] 1>/dev/null 2>&1 && echo ok"
	_, stdout, _ := ExeCmd(detect_str)
	if strings.TrimSpace(stdout) == "ok" {
		return true
	}
	detect_str = "strings /sbin/init | grep -q \"/lib/systemd\" && echo ok"
	_, stdout, _ = ExeCmd(detect_str)
	if strings.TrimSpace(stdout) == "ok" {
		return true
	}
	return false
}

func IsUpstartLinux() bool {
	detect_str := "[[ `/sbin/init --version` =~ upstart ]] 1>/dev/null 2>&1 && echo ok"
	_, stdout, _ := ExeCmd(detect_str)
	if strings.TrimSpace(stdout) == "ok" {
		return true
	}
	detect_str = "strings /sbin/init | grep -q \"upstart\" && echo ok"
	_, stdout, _ = ExeCmd(detect_str)
	if strings.TrimSpace(stdout) == "ok" {
		return true
	}
	return false
}

func IsSysVLinux() bool {
	detect_str := "[[ -f /etc/init.d/cron && ! -h /etc/init.d/cron ]] 1>/dev/null 2>&1 && echo ok"
	_, stdout, _ := ExeCmd(detect_str)
	if strings.TrimSpace(stdout) == "ok" {
		return true
	}
	detect_str = "[[ -f /etc/init.d/crond && ! -h /etc/init.d/cron ]] 1>/dev/null 2>&1 && echo ok"
	_, stdout, _ = ExeCmd(detect_str)
	if strings.TrimSpace(stdout) == "ok" {
		return true
	}
	detect_str = "strings /sbin/init | grep -q \"sysvinit\"  && echo ok"
	_, stdout, _ = ExeCmd(detect_str)
	if strings.TrimSpace(stdout) == "ok" {
		return true
	}
	return false
}

func IsServiceExist(ServiceName string) bool {
	var detect_str string
	if IsSystemdLinux() {
		detect_str = "systemctl | grep " + ServiceName + ".service"
	} else if IsUpstartLinux() {
		detect_str = "initctl list | grep " + ServiceName
	} else if IsSysVLinux() {
		detect_str = "service --status-all | grep " + ServiceName
	} else {
		return false
	}
	_, stdout, _ := ExeCmd(detect_str)
	if strings.Contains(stdout, ServiceName) {
		return true
	} else {
		return false
	}
}

func IsServiceRunning(ServiceName string) bool {
	if IsSystemdLinux() {
		detect_str := "systemctl is-active " + ServiceName + ".service"
		_, stdout, _ := ExeCmd(detect_str)
		if strings.Contains(stdout, "active") {
			return true
		} else {
			return false
		}
	} else if IsUpstartLinux() {
		detect_str := "initctl status " + ServiceName
		_, stdout, _ := ExeCmd(detect_str)
		if strings.Contains(stdout, "start/running") {
			return true
		} else {
			return false
		}
	} else if IsSysVLinux() {
		detect_str := "service " + ServiceName + " status"
		_, stdout, _ := ExeCmd(detect_str)
		if strings.Contains(stdout, "Running") {
			return true
		} else {
			return false
		}
	} else {
		return false
	}
}

func StartService(ServiceName string) error {
	if IsServiceRunning(ServiceName) {
		return nil
	}
	var err error
	if IsSystemdLinux() {
		err, _, _ = ExeCmd("systemctl start " + ServiceName + ".service")
	} else if IsUpstartLinux() {
		err, _, _ = ExeCmd("initctl start " + ServiceName)
	} else if IsSysVLinux() {
		err, _, _ = ExeCmd("service " + ServiceName + " start")
	} else {
		return errors.New("Unkown System")
	}
	return err
}

func StopService(ServiceName string) error {
	if !IsServiceRunning(ServiceName) {
		return nil
	}
	var err error
	if IsSystemdLinux() {
		err, _, _ = ExeCmd("systemctl stop " + ServiceName + ".service")
	} else if IsUpstartLinux() {
		err, _, _ = ExeCmd("initctl stop " + ServiceName)
	} else if IsSysVLinux() {
		err, _, _ = ExeCmd("service " + ServiceName + " stop")
	} else {
		return errors.New("Unkown System")
	}
	return err
}