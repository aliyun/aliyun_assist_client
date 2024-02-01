//go:build windows

package util

import (
	"path/filepath"
)

// CheckFileIsExecutable check if the file is executable by file extension
func CheckFileIsExecutable(fileName string) bool {
	absPath, err := filepath.Abs(fileName)
	if err != nil {
		return false
	}
	ext := filepath.Ext(absPath)
	return ext == ".exe" || ext == ".ps1" || ext == ".bat" || ext == ".cmd"
}
