package update

import (
	"debug/elf"
	"fmt"
)

func ValidateExecutable(executablePath string) error {
	executable, err := elf.Open(executablePath)
	if err != nil {
		return fmt.Errorf("%w: %s", ErrInvalidELF, err.Error())
	}

	if executable.FileHeader.OSABI != elf.ELFOSABI_NONE &&
		executable.FileHeader.OSABI != elf.ELFOSABI_LINUX {
		return fmt.Errorf("%w: %s", ErrELFUnsupportedOSABI, executable.FileHeader.OSABI.String())
	}

	if executable.FileHeader.Machine != elf.EM_AARCH64 {
		return fmt.Errorf("%w: %s", ErrELFUnsupportedArchitecture, executable.FileHeader.Machine.String())
	}

	return nil
}
