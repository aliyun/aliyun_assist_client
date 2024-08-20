package kickvmhandle

import (
	"testing"
	"fmt"

	"github.com/aliyun/aliyun_assist_client/agent/version"
	"github.com/stretchr/testify/assert"
)

func TestCompareWithCurVersion(t *testing.T) {
	originVersion := version.AssistVersion
	defer func() {
		version.AssistVersion = originVersion
	}()

	tests := []struct{
		newVersion string
		currentVersion string
		expectErr bool
		expectRes int
	}{
		// lower
		{
			newVersion: "2.1.3.99",
			currentVersion: "2.1.3.100",
			expectErr: false,
			expectRes: -1,
		},
		{
			newVersion: "2.1.2.100",
			currentVersion: "2.1.3.100",
			expectErr: false,
			expectRes: -1,
		},
		// equal
		{
			newVersion: "2.1.3.100",
			currentVersion: "2.1.3.100",
			expectErr: false,
			expectRes: 0,
		},
		// higher
		{
			newVersion: "2.1.3.101",
			currentVersion: "2.1.3.100",
			expectErr: false,
			expectRes: 1,
		},
		{
			newVersion: "2.1.4.100",
			currentVersion: "2.1.3.100",
			expectErr: false,
			expectRes: 1,
		},
		// error
		{
			// main failed not match
			newVersion: "2.2.3.100",
			currentVersion: "2.1.3.100",
			expectErr: true,
			expectRes: 1,
		},
		{
			// invalid format
			newVersion: "2.1.3",
			currentVersion: "2.1.3.100",
			expectErr: true,
			expectRes: 1,
		},
	}
	for _, tt := range tests {
		version.AssistVersion = tt.currentVersion
		res, err := compareWithCurVersion(tt.newVersion)
		fmt.Println(res, err)
		if tt.expectErr {
			assert.NotNil(t, err)
		} else {
			assert.Equal(t, tt.expectRes, res)
		}
	}
}