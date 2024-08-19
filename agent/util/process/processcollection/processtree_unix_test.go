//go:build !windows

package processcollection

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestProbeProcessMap(t *testing.T) {
	childProcessMap, processNameMap, err := probeProcessMap()
	assert.Equal(t, nil, err)
	assert.NotEqual(t, 0, len(childProcessMap))
	assert.NotEqual(t, 0, len(processNameMap))
}

func TestKillRecur(t *testing.T) {
	childProcessMap := map[int32][]int32{
		9999: []int32{9998},
	}
	processNameMap := map[int32]string{
		9999: "parent process",
		9998: "child process",
	}
	err := killRecur(9999, childProcessMap, processNameMap)
	fmt.Println(err)
	assert.NotEqual(t, nil, err)
}
