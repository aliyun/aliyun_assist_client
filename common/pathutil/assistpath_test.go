package pathutil

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)


func TestGetCurrentPath(t *testing.T) {
	path,err := GetCurrentPath()
	assert.DirExists(t, path)
	assert.Equal(t, true, err==nil)
}

func TestEnvSet(t *testing.T) {
	os.Setenv("path1", "test")
	path := os.Getenv("path1")
	assert.Equal(t, path, "test")
}

