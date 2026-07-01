//go:build linux || freebsd
// +build linux freebsd

package pluginmanager

import (
	"syscall"
	"testing"

	gomonkey "github.com/agiledragon/gomonkey/v2"
	"github.com/stretchr/testify/assert"

	"github.com/aliyun/aliyun_assist_client/agent/pluginmodel"
	"github.com/aliyun/aliyun_assist_client/agent/util/osutil"
)

func TestGetArch(t *testing.T) {
    testCases := []struct {
        name        string
        mockOutput  string
        mockErr     error
        expected    string
        expectedRaw string
    }{
        // Well-known architectures
        {"x86_64", "x86_64", nil, pluginmodel.ARCH_64, "x86_64"},
        {"aarch64", "aarch64", nil, pluginmodel.ARCH_ARM, "aarch64"},
        {"i686", "i686", nil, pluginmodel.ARCH_32, "i686"},

        // Edge cases
        {"MixedCase", "AaRcH64", nil, pluginmodel.ARCH_ARM, "aarch64"},

        // Erroneous cases
        {"ErrorCase", "", syscall.EIO, pluginmodel.ARCH_UNKNOWN, ""},
        {"UnknownArch", "ppc64le", nil, pluginmodel.ARCH_UNKNOWN, "ppc64le"},
    }

    for _, tc := range testCases {
        t.Run(tc.name, func(t *testing.T) {
            guard := gomonkey.ApplyFunc(osutil.GetUnameMachine, func() (string, error) {
				return tc.mockOutput, tc.mockErr
			})
            defer guard.Reset()

            formatArch, rawArch := GetArch()
            assert.Equal(t, tc.expected, formatArch)
            assert.Equal(t, tc.expectedRaw, rawArch)
        })
    }
}
