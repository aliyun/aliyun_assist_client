//+build !windows

package util

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestExeCmdWithContext(t *testing.T) {
	tests := []struct{
		cmd string
		timeout int
		willTimeout bool
	}{
		{
			cmd: "sleep 1",
			timeout: 2,
			willTimeout: false,
		},
		{
			cmd: "sleep 2",
			timeout: 1,
			willTimeout: true,
		},
	}
	for _, tt := range tests {
		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(tt.timeout) * time.Second)
		err, _, _ := ExeCmdWithContext(ctx, tt.cmd)
		assert.Equal(t, tt.willTimeout, err != nil)
		cancel()
	}
}

func TestExeCmd(t *testing.T) {
	tests := []struct{
		cmd string
		expectedOut string
		expectedErr string
	}{
		{
			cmd: "sleep 1",
		},
		{
			cmd: "echo -n '123'; echo -n 'abc'>&2; sleep 1; echo -n '456'; echo -n 'def'>&2",
			expectedOut: "123456",
			expectedErr: "abcdef",
		},
	}
	for _, tt := range tests {
		err, stdout, stderr := ExeCmd(tt.cmd)
		assert.Nil(t, err)
		assert.Equal(t, tt.expectedOut, stdout)
		assert.Equal(t, tt.expectedErr, stderr)
	}
}