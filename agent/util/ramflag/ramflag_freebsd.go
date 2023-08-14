package ramflag

import (
	"errors"
	"os"
	"strings"

	"github.com/fabiokung/shm"
)

// The path need begin	with a slash (`/') character.
func IsExist(name string) (bool, error) {
	if !strings.HasPrefix(name, "/") {
		name = "/" + name
	}
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
	if !strings.HasPrefix(name, "/") {
		name = "/" + name
	}
	shmFile, err := shm.Open(name, os.O_RDWR | os.O_CREATE | os.O_EXCL, 0600)
	if err != nil {
		return err
	}

	defer shmFile.Close()
	return nil
}

func Delete(name string) error {
	if !strings.HasPrefix(name, "/") {
		name = "/" + name
	}
	return shm.Unlink(name)
}
