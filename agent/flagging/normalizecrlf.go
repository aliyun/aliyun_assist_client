package flagging

import (
	"go.uber.org/atomic"
)

const (
	disableNormalizeCRLFFlagFilename = "disable_normalize_crlf"
	disableNormalizeCRLFFlagName     = "Disable CRLF-normalization"
)

var (
	_normalizingCRLFDisabled atomic.Bool
)

func IsNormalizingCRLFDisabled() bool {
	return _normalizingCRLFDisabled.Load()
}

func DetectNormalizingCRLFDisabled() (bool, error) {
	return detectFlagSet(disableNormalizeCRLFFlagName, disableNormalizeCRLFFlagFilename, &_normalizingCRLFDisabled)
}
