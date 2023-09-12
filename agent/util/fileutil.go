package util

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/aliyun/aliyun_assist_client/common/pathutil"
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

// 将 srcPath/目录下的文件拷贝到 destPath/目录， srcPath 和 destPath 需要是已经存在的路径，如果 destPath 下存在同名文件将会被覆盖
func CopyDir(srcPath string, destPath string) error {
	if !filepath.IsAbs(srcPath) {
		srcPath, _ = filepath.Abs(srcPath)
	}
	if !filepath.IsAbs(destPath) {
		destPath, _ = filepath.Abs(destPath)
	}
 	if srcInfo, err := os.Stat(srcPath); err != nil {
 		return err
 	} else {
 		if !srcInfo.IsDir() {
			return errors.New(fmt.Sprintf("'%s' is not a valid dir", srcPath))
		}
 	}
	if destInfo, err := os.Stat(destPath); err != nil {
		return err
	} else {
		if !destInfo.IsDir() {
			e := errors.New(fmt.Sprintf("'%s' is not a valid dir", destPath))
			return e
		}
	}
	err := filepath.Walk(srcPath, func(path string, f os.FileInfo, err error) error {
		if f == nil {
			return err
		}
		destNewPath := strings.Replace(path, srcPath, destPath, -1)
		if !f.IsDir() {
			copyFile(path, destNewPath)
		} else {
			pathutil.MakeSurePath(destPath)
		}
		return nil
	})
	return err
}

func copyFile(src, dest string) (err error) {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()
	idx := strings.LastIndex(dest, string(filepath.Separator))
	destdir := dest
	if idx > 0 {
		destdir = dest[:idx+1]
	}
	pathutil.MakeSurePath(destdir)
	dstFile, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer dstFile.Close()
	_, err = io.Copy(dstFile, srcFile)
	return err
}