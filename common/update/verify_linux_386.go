package update

import (
	"debug/elf"
	"fmt"

	"github.com/aliyun/aliyun_assist_client/agent/util/osutil"
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

	// There are still some x86 instances running on our cloud and that is why
	// agent has to be compiled with GOARCH=386 when GOOS=linux. Much careful
	// handling these instances during updating procedures is neccessary. Thus
	// machine architecture MUST be obtained before further disscussion.
	unameMachine, err := osutil.GetUnameMachine()
	if err != nil {
		return err
	}
	// There are also many names or aliases for machine architectures. Try our
	// best to match all possible names for correct handling. For possible
	// values, see https://stackoverflow.com/a/45125525 for a incomplete list.
	if unameMachine == "i386" || unameMachine == "i686" || unameMachine == "x86" {
		// 32-bit x86 architecture only accepts 32-bit x86 executables
		if executable.FileHeader.Machine != elf.EM_386 {
			return fmt.Errorf("%w: %s", ErrELFUnsupportedArchitecture, executable.FileHeader.Machine.String())
		}
	} else {
		// Otherwise, looks like happily running on x86_64 architecture
		if executable.FileHeader.Machine != elf.EM_386 &&
			executable.FileHeader.Machine != elf.EM_X86_64 {
			return fmt.Errorf("%w: %s", ErrELFUnsupportedArchitecture, executable.FileHeader.Machine.String())
		}
	}

	return nil
}
