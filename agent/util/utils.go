package util

import (
	"archive/zip"
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/aliyun/aliyun_assist_client/agent/util/machineid"
)

var G_IsWindows bool
var verbose_mode = false

func init() {
	if runtime.GOOS == "windows" {
		G_IsWindows = true
	} else if runtime.GOOS == "linux" {
		G_IsWindows = false
	} else {

	}
}

func IsVerboseMode() bool {
	return verbose_mode
}

func SetVerboseMode(mode bool)  {
	verbose_mode = mode
}

func GetMachineID() (string, error) {
	path, _ := GetHybridPath()
	path += "/machine-id"
	if CheckFileIsExist(path) {
		content, _ := ioutil.ReadFile(path)
		cached_id := string(content)
		return cached_id, nil
	}
	return machineid.GetMachineID()
}

/*
func HttpDownlod(url string, FilePath string) error {
	res, err := http.Get(url)
	if err != nil {
		return err
	}
	f, err := os.Create(FilePath)
	defer f.Close()
	if err != nil {
		return err
	}
	_, err = io.Copy(f, res.Body)
	return err
}*/

func Unzip(zipFile string, destDir string) error {
	zipReader, err := zip.OpenReader(zipFile)
	if err != nil {
		return err
	}
	defer zipReader.Close()

	for _, f := range zipReader.File {
		fpath := filepath.Join(destDir, f.Name)
		if f.FileInfo().IsDir() {
			os.MkdirAll(fpath, os.ModePerm)
		} else {
			if err = os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
				return err
			}

			inFile, err := f.Open()
			if err != nil {
				return err
			}

			outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
			if err != nil {
				inFile.Close()
				return err
			}

			_, err = io.Copy(outFile, inFile)
			outFile.Close()
			inFile.Close()
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func ComputeMd5(filePath string) (string, error) {
	var result []byte
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()
	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(result)), nil
}

func ComputeStrMd5(str string) string {
	h := md5.New()
	h.Write([]byte(str))
	return hex.EncodeToString(h.Sum(nil))
}

func ComputeBinMd5(bin []byte) string {
	h := md5.New()
	h.Write(bin)
	return hex.EncodeToString(h.Sum(nil))
}

func FileExist(path string) bool {
	_, err := os.Lstat(path)
	return !os.IsNotExist(err)
}

func IsDirectory(path string) bool {
	fi, err := os.Stat(path)
	if err != nil {
		return false
	}
	if fi.Mode().IsDir() {
		return true
	}
	return false
}

func IsFile(path string) bool {
	fi, err := os.Stat(path)
	if err != nil {
		return false
	}
	if fi.Mode().IsRegular() {
		return true
	}
	return false
}

func ExeCmd(cmd string) (error, string, string) {
	var command *exec.Cmd
	if G_IsWindows {
		command = exec.Command("cmd", "/c", cmd)
	} else {
		command = exec.Command("sh", "-c", cmd)
	}
	var outInfo bytes.Buffer
	var errInfo bytes.Buffer
	command.Stdout = &outInfo
	command.Stderr = &errInfo
	err := command.Run()
	if nil != err {
		return err, "", ""
	}

	return nil, outInfo.String(), errInfo.String()
}

func IsSystemdLinux() bool {
	if G_IsWindows {
		return false
	}
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
	if G_IsWindows {
		return false
	}
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
	if G_IsWindows {
		return false
	}
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
	if G_IsWindows {
		detect_str = "sc query | findstr " + ServiceName
	} else {
		if IsSystemdLinux() {
			detect_str = "systemctl | grep " + ServiceName + ".service"
		} else if IsUpstartLinux() {
			detect_str = "initctl list | grep " + ServiceName
		} else if IsSysVLinux() {
			detect_str = "service --status-all | grep " + ServiceName
		} else {
			return false
		}
	}
	_, stdout, _ := ExeCmd(detect_str)
	if strings.Contains(stdout, ServiceName) {
		return true
	} else {
		return false
	}
}

func IsServiceRunning(ServiceName string) bool {
	if G_IsWindows {
		detect_str := "sc query " + ServiceName
		_, stdout, _ := ExeCmd(detect_str)
		if strings.Contains(stdout, " RUNNING") {
			return true
		} else {
			return false
		}
	} else {
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
}

func StartService(ServiceName string) error {
	if IsServiceRunning(ServiceName) {
		return nil
	}
	var err error
	if G_IsWindows {
		err, _, _ = ExeCmd("net start " + ServiceName)
	} else {
		if IsSystemdLinux() {
			err, _, _ = ExeCmd("systemctl start " + ServiceName + ".service")
		} else if IsUpstartLinux() {
			err, _, _ = ExeCmd("initctl start " + ServiceName)
		} else if IsSysVLinux() {
			err, _, _ = ExeCmd("service " + ServiceName + " start")
		} else {
			return errors.New("Unkown System")
		}
	}
	return err
}

func StopService(ServiceName string) error {
	if !IsServiceRunning(ServiceName) {
		return nil
	}
	var err error
	if G_IsWindows {
		err, _, _ = ExeCmd("net stop " + ServiceName)
	} else {
		if IsSystemdLinux() {
			err, _, _ = ExeCmd("systemctl stop " + ServiceName + ".service")
		} else if IsUpstartLinux() {
			err, _, _ = ExeCmd("initctl stop " + ServiceName)
		} else if IsSysVLinux() {
			err, _, _ = ExeCmd("service " + ServiceName + " stop")
		} else {
			return errors.New("Unkown System")
		}
	}
	return err
}

func HasCmdInLinux(cmd string) bool {
	err, _, _ := ExeCmd("which " + cmd)
	if err != nil {
		return false
	}
	return true
}
