package shm

import (
	"os"
	"path/filepath"
)

func Open(regionName string, flags int, perm os.FileMode) (*os.File, error) {
	filename := filepath.Join("/dev/shm", regionName)
	file, err := os.OpenFile(filename, flags, perm)
	if err != nil {
		return nil, err
	}
	return file, nil
}

func Unlink(regionName string) error {
	path := regionName
	if !filepath.IsAbs(path) {
		path = filepath.Join("/dev/shm", regionName)
	}
	return os.Remove(path)
}
