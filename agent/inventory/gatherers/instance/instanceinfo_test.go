package instance

import (
	"testing"
	"fmt"

	"github.com/stretchr/testify/assert"
)

func TestGetInstanceInfo(t *testing.T) {
	info, _ := GetInstanceInfo()

	fmt.Println(info)

	assert.NotNil(t, info)
}
