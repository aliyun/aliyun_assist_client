package acspluginmanager

import (
	"bytes"
	"io"
	"io/fs"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"

	. "github.com/aliyun/aliyun_assist_client/agent/pluginmanager"
	jsoniter "github.com/aliyun/aliyun_assist_client/agent/pluginmanager/acspluginmanager/thirdparty/json-iterator/go"
	"github.com/aliyun/aliyun_assist_client/agent/pluginmanager/acspluginmanager/thirdparty/json-iterator/go/extra"
	"github.com/aliyun/aliyun_assist_client/agent/util"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

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
	jsonEncoder.SetIndent("", "    ")
	err = jsonEncoder.Encode(obj)
	if err != nil {
		return
	}
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

// DeleteSlice 删除指定元素。
func DeletePluginInfoByIdx(pluginInfos []PluginInfo, idx int) []PluginInfo {
    j := 0
    for i, p := range pluginInfos {
        if i != idx {
            pluginInfos[j] = p
            j++
        }
    }
    return pluginInfos[:j]
}
