package machineid

import (
	"fmt"
	"runtime"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestMachineId(t *testing.T) {
	value1,err1 := machineID()
	value2,err2:= machineID()
	assert.Equal(t, value1, value2)
	assert.Equal(t, err1, nil)
	assert.Equal(t, err2, nil)
	u4 := uuid.New()
	fmt.Println(u4.String()) // a0d99f20-1dd1-459b-b516-dfeca4005203
	fmt.Println(runtime.GOARCH)
}