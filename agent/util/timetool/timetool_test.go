package timetool

import (
	"testing"
	"time"
	
	"github.com/stretchr/testify/assert"
)

func TestApiTimeFormat(t *testing.T) {
	var epoch time.Time = time.Unix(0, 0)
	assert.Equal(t, "1970-01-01T00:00:00Z", ApiTimeFormat(epoch))

    cst, _ := time.LoadLocation("Asia/Shanghai")
	epoch2, _ := time.ParseInLocation("2006-01-02 15:04:05", "1970-01-01 08:00:00", cst)
	assert.Equal(t, "1970-01-01T00:00:00Z", ApiTimeFormat(epoch2))
}

func TestParseApiTime(t *testing.T) {
	epoch, err := ParseApiTime("1970-01-01T00:00:00Z")
	assert.Nil(t, err)
	assert.Equal(t, 0, int(epoch.Unix()))
}