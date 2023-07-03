package fuzzyjson

import (
	"bytes"
	"io/ioutil"

	jsoniter "github.com/aliyun/aliyun_assist_client/thirdparty/json-iterator/go"
	"github.com/aliyun/aliyun_assist_client/thirdparty/json-iterator/go/extra"
)

var (
	json = jsoniter.ConfigCompatibleWithStandardLibrary
)

func init() {
	// 插件版本号字段定义为string，但是一些插件该字段是int。这个开关打开后能够把json中的int float类型转换成string
	extra.RegisterFuzzyDecoders()
}

func UnmarshalFile(filePath string, dest interface{}) (content []byte, err error) {
	content, err = ioutil.ReadFile(filePath)
	if err != nil {
		return
	}
	err = json.Unmarshal(content, dest)
	return
}

// Unmarshal unmarshals the content in string format to an object.
func Unmarshal(jsonContent string, dest interface{}) (err error) {
	content := []byte(jsonContent)
	err = json.Unmarshal(content, dest)
	return
}

// Marshal marshals an object to a json string.
// Returns empty string if marshal fails.
func Marshal(obj interface{}) (result string, err error) {
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
