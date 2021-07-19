package update

import "errors"

var (
	ErrInvalidELF = errors.New("Invalid ELF-format executable")
	ErrELFUnsupportedOSABI = errors.New("Unsupported OSABI identifier of ELF executable")
	ErrELFUnsupportedArchitecture = errors.New("Unsupported architecture identifier of ELF executable")

	ErrInvalidPE = errors.New("Invalid PE-format executable")
	ErrPEUnsupportedArchitecture = errors.New("Unsupported architecture identifier of PE executable")
)
