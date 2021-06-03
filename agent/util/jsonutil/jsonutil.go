package jsonutil

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
)

// jsonFormat json formatIndent
const jsonFormat = "  "

// Indent indents a json string.
func Indent(jsonStr string) string {
	var dst bytes.Buffer
	json.Indent(&dst, []byte(jsonStr), "", jsonFormat)
	return string(dst.Bytes())
}

// Remarshal marshals an object to Json then parses it back to another object.
// This is useful for example when we want to go from map[string]interface{}
// to a more specific struct type or if we want a deep copy of the object.
func Remarshal(obj interface{}, remarshalledObj interface{}) (err error) {
	b, err := json.Marshal(obj)
	if err != nil {
		return
	}
	err = json.Unmarshal(b, remarshalledObj)
	if err != nil {
		return
	}
	return nil
}

// Marshal marshals an object to a json string.
// Returns empty string if marshal fails.
func Marshal(obj interface{}) (result string, err error) {
	bf := bytes.NewBuffer([]byte{})
	jsonEncoder := json.NewEncoder(bf)
	jsonEncoder.SetEscapeHTML(false)
	jsonEncoder.Encode(obj)
	result = bf.String()
	//var resultB []byte
	//resultB, err = json.Marshal(obj)
	//if err != nil {
	//	return
	//}
	//result = string(resultB)
	return
}

// UnmarshalFile reads the content of a file then Unmarshals the content to an object.
func UnmarshalFile(filePath string, dest interface{}) (err error) {
	content, err := ioutil.ReadFile(filePath)
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

// MarshalIndent is like Marshal but applies Indent to format the output.
// Returns empty string if marshal fails
func MarshalIndent(obj interface{}) (result string, err error) {
	var resultsByte []byte
	// Make sure the output file keeps formal json format
	resultsByte, err = json.MarshalIndent(obj, "", jsonFormat)
	if err != nil {
		return
	}
	result = string(resultsByte)
	return
}
