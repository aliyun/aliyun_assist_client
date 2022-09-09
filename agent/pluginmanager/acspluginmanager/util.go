package acspluginmanager

import (
	"bytes"
	"os/exec"
	"io"
	"io/fs"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/aliyun/aliyun_assist_client/agent/log"
	jsoniter "github.com/aliyun/aliyun_assist_client/agent/pluginmanager/acspluginmanager/thirdparty/json-iterator/go"
	"github.com/aliyun/aliyun_assist_client/agent/pluginmanager/acspluginmanager/thirdparty/json-iterator/go/extra"
	"github.com/aliyun/aliyun_assist_client/agent/util"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

const (
	ARCH_64 = "x64"
	ARCH_32 = "x86"
	ARCH_ARM = "arm"
	ARCH_UNKNOWN = "unknown"
)

func init() {
	// 插件版本号字段定义为string，但是一些插件该字段是int。这个开关打开后能够把json中的int float类型转换成string
	extra.RegisterFuzzyDecoders()
}

func unmarshalFile(filePath string, dest interface{}) (content []byte, err error) {
	content, err = ioutil.ReadFile(filePath)
	if err != nil {
		return
	}
	err = json.Unmarshal(content, dest)
	return
}

// Unmarshal unmarshals the content in string format to an object.
func unmarshal(jsonContent string, dest interface{}) (err error) {
	content := []byte(jsonContent)
	err = json.Unmarshal(content, dest)
	return
}

// Marshal marshals an object to a json string.
// Returns empty string if marshal fails.
func marshal(obj interface{}) (result string, err error) {
	bf := bytes.NewBuffer([]byte{})
	jsonEncoder := json.NewEncoder(bf)
	jsonEncoder.SetEscapeHTML(false)
	jsonEncoder.Encode(obj)
	result = bf.String()
	return
}

type WinDir string

// 针对windows的路径模式定义的FileSystem，参照net/http.Dir
func (d WinDir) Open(name string) (http.File, error) {
	if len(name) > 0 && name[0] == '/' {
		name = name[1:]
	}
	fullName := filepath.FromSlash(path.Clean(name))
	f, err := os.Open(fullName)
	if err != nil {
		return nil, mapDirOpenError(err, fullName)
	}
	return f, nil
}

func mapDirOpenError(originalErr error, name string) error {
	if os.IsNotExist(originalErr) || os.IsPermission(originalErr) {
		return originalErr
	}

	parts := strings.Split(name, string(filepath.Separator))
	for i := range parts {
		if parts[i] == "" {
			continue
		}
		fi, err := os.Stat(strings.Join(parts[:i+1], string(filepath.Separator)))
		if err != nil {
			return originalErr
		}
		if !fi.IsDir() {
			return fs.ErrNotExist
		}
	}
	return originalErr
}

func FileProtocolDownload(url, filePath string) error {
	t := util.GetHTTPTransport()
	if runtime.GOOS == "windows" {
		t.RegisterProtocol("file", http.NewFileTransport(WinDir("")))
	} else {
		t.RegisterProtocol("file", http.NewFileTransport(http.Dir("/")))
	}
	c := &http.Client{Transport: t}
	res, err := c.Get(url)
	if err != nil {
		return err
	}
	f, err := os.Create(filePath)
	defer f.Close()
	if err != nil {
		return err
	}
	_, err = io.Copy(f, res.Body)
	return err
}

func getArch() (formatArch string, rawArch string) {
	defer func() {
		log.GetLogger().Errorf("Get Arch: formatArch[%s] rawArch[%s]: ", formatArch, rawArch)
	}()
	formatArch = ARCH_UNKNOWN
	if runtime.GOOS == "windows" {
		// 云助手的windows版架构只有amd64的
		formatArch = ARCH_64
		rawArch = "windows arch"
		return
	} else {
		// 执行 uname -m 获得系统的架构名称
		var outInfo bytes.Buffer
		cmd := exec.Command("uname", "-m")

		cmd.Stdout = &outInfo
		if err := cmd.Run(); err != nil {
			log.GetLogger().Errorln("Get Arch err: ", err.Error())
			return
		}
		arch := outInfo.String()
		arch = strings.TrimSpace(arch)
		arch = strings.ToLower(arch)
		rawArch = arch

		if strings.Contains(arch, "aarch") || strings.Contains(arch, "arm"){ // arm: aarch arm
			formatArch = ARCH_ARM
		} else if strings.Contains(arch, "386") || strings.Contains(arch, "686") { // x86: i386 i686
			formatArch = ARCH_32
		} else if  arch == "x86_64" { // x64: x86_64
			formatArch = ARCH_64
		} else {
			log.GetLogger().Errorln("Get Arch: unknown arch: ", arch)
			formatArch = ARCH_UNKNOWN
		}
	}
	return
}