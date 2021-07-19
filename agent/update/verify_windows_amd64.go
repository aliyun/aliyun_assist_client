package update

import (
	"debug/pe"
	"fmt"
)

func ValidateExecutable(executablePath string) error {
	executable, err := pe.Open(executablePath)
	if err != nil {
		return fmt.Errorf("%w: %s", ErrInvalidPE, err.Error())
	}

	if executable.FileHeader.Machine != pe.IMAGE_FILE_MACHINE_AMD64 {
		return fmt.Errorf("%w: %x", ErrPEUnsupportedArchitecture, executable.FileHeader.Machine)
	}

	return nil
}
