package fileutil

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func CheckFileIsExist(filename string) bool {
	var exist = true
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		exist = false
	}
	return exist
}

func CheckDirectoryIsExist(dirname string) bool {
	var exist = true
	if fi, err := os.Stat(dirname); os.IsNotExist(err) {
		exist = false
	} else {
		if !fi.IsDir() {
			exist = false
		}
	}
	return exist
}

func WriteStringToFile(path string, content string) error {
	var d1 = []byte(content)
	err := os.WriteFile(path, d1, 0666) //写入文件(字节数组)
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
			os.MkdirAll(destPath, os.ModePerm)
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
	os.MkdirAll(destdir, os.ModePerm)
	dstFile, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer dstFile.Close()
	_, err = io.Copy(dstFile, srcFile)
	return err
}

func WriteAndSyncFile(name string, data []byte, perm os.FileMode) error {
	f, err := os.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, perm)
	if err != nil {
		return err
	}

	_, err = f.Write(data)

	// Sync file from memory to disk, and override error indication only when
	// there was no error before
	if err1 := f.Sync(); err1 != nil && err == nil {
		err = err1
	}

	// Close file, and override error indication only when there was no error
	if err2 := f.Close(); err2 != nil && err == nil {
		err = err2
	}
	return err
}
