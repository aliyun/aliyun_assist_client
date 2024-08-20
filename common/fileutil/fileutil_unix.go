//go:build !windows

package fileutil

import (
	"os"
	"path/filepath"
)

func CheckFileIsExecutable(fileName string) bool {
	absPath, err := filepath.Abs(fileName)
	if err != nil {
		return false
	}

	fileInfo, err := os.Stat(absPath)
	if err != nil {
		return false
	}

	return fileInfo.Mode().Perm()&0100 != 0
}
