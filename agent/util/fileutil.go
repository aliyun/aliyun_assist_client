package util

import (
	"io/ioutil"
	"os"
)

func CheckFileIsExist(filename string) bool {
	var exist = true
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		exist = false
	}
	return exist
}

func WriteStringToFile(path string, content string) error {
	var d1 = []byte(content)
	err := ioutil.WriteFile(path, d1, 0666) //写入文件(字节数组)
	return err
}
