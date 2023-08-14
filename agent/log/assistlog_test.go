package log

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetLogger(t *testing.T) {
	InitLog("test", "", false)
	assert.NotNil(t, GetLogger())
}
