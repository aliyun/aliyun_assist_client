// +build linux freebsd

package ramflag

import (
	"errors"
	"os"

	"github.com/fabiokung/shm"
)

func IsExist(name string) (bool, error) {
	shmFile, err := shm.Open(name, os.O_RDWR, 0600)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		} else {
			return false, err
		}
	}

	defer shmFile.Close()
	return true, nil
}

func Create(name string) error {
	shmFile, err := shm.Open(name, os.O_RDWR | os.O_CREATE | os.O_EXCL, 0600)
	if err != nil {
		return err
	}

	defer shmFile.Close()
	return nil
}

func Delete(name string) error {
	return shm.Unlink(name)
}
