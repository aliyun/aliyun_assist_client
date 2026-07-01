//go:build linux || darwin || freebsd
// +build linux darwin freebsd

package pluginmodel

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHostArchitecture2Model(t *testing.T) {
    testCases := []struct {
        name        string
        arch        string
        wantResult  string
        wantBool    bool
    }{
        // ARM cases
        {"aarch64", "aarch64", ARCH_ARM, true},
        {"arm", "arm", ARCH_ARM, true},
        {"armv7l", "armv7l", ARCH_ARM, true},
        {"arm64", "arm64", ARCH_ARM, true},
        {"ARM", "ARM", ARCH_UNKNOWN, false}, // unsupported uppercase word

        // x86 cases
        {"386", "386", ARCH_32, true},
        {"i386", "i386", ARCH_32, true},
        {"686", "686", ARCH_32, true},
        {"i686", "i686", ARCH_32, true},
        {"x86", "x86", ARCH_UNKNOWN, false}, // not recognized yet
        {"X86", "X86", ARCH_UNKNOWN, false}, // unsupported uppercase word

        // x86-64 cases
        {"x86_64", "x86_64", ARCH_64, true},
        {"amd64", "amd64", ARCH_64, true},
        {"AMD64", "AMD64", ARCH_UNKNOWN, false}, // unsupported uppercase word

        // unsupported architectures
        {"empty", "", ARCH_UNKNOWN, false},
        {"ppc64le", "ppc64le", ARCH_UNKNOWN, false},
        {"s390x", "s390x", ARCH_UNKNOWN, false},
        {"mips64", "mips64", ARCH_UNKNOWN, false},
    }

    for _, tc := range testCases {
        t.Run(tc.name, func(t *testing.T) {
            modelArch, ok := HostArchitecture2Model(tc.arch)
            assert.Equalf(t, tc.wantResult, modelArch, `HostArchitecture2Model("%s")`, tc.arch)
            assert.Equalf(t, tc.wantBool, ok, `HostArchitecture2Model("%s")`, tc.arch)
        })
    }
}
