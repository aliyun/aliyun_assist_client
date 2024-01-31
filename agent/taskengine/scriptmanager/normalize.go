package scriptmanager

import (
	"strings"
)

var (
	_CRLF = []byte{'\r', '\n'}
)

func NormalizeCRLF(orig string) string {
	var normalized strings.Builder

	begin := 0
	for i := 0; i < len(orig); i++ {
		if orig[i] == '\n' {
			if i == begin {
				normalized.Write(_CRLF)
				begin = i + 1
			} else if orig[i - 1] != '\r' {
				normalized.WriteString(orig[begin:i])
				normalized.Write(_CRLF)
				begin = i + 1
			}
		}
	}
	if begin < len(orig) {
		normalized.WriteString(orig[begin:])
	}

	return normalized.String()
}
