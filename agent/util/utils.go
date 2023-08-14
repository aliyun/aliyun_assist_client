package util

import (
	"crypto/md5"
	"encoding/hex"
	"io"
	"io/ioutil"
	"os"
	"runtime"

	"github.com/aliyun/aliyun_assist_client/agent/util/machineid"
	
)

var G_IsWindows bool
var verbose_mode = false

func init() {
	if runtime.GOOS == "windows" {
		G_IsWindows = true
	} else if runtime.GOOS == "linux" || runtime.GOOS == "freebsd" {
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

func HasCmdInLinux(cmd string) bool {
	err, _, _ := ExeCmd("which " + cmd)
	if err != nil {
		return false
	}
	return true
}
