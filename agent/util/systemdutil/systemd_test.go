package systemdutil

import (
	"reflect"
	"testing"
	"unsafe"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/aliyun/aliyun_assist_client/thirdparty/service"
	"github.com/stretchr/testify/assert"
)

func TestIsRunningSystemd(t *testing.T) {
	resetState := func() { //reset isRunningSystemdOnce
		isRunningSystemd = false
		v := reflect.ValueOf(&isRunningSystemdOnce).Elem()
		done := v.FieldByName("done")
		if done.IsValid() && done.CanSet() {
			done.SetUint(0)
		} else {
			// fallback: use unsafe (if field name changed)
			addr := v.UnsafeAddr()
			*(*uint32)(unsafe.Pointer(addr)) = 0
		}
	}

	t.Run("returns true when IsSystemd returns true", func(t *testing.T) {
		called := false
		defer gomonkey.ApplyFunc(service.IsSystemd, func() bool {
			called = true
			return true
		}).Reset()
		isRunningSystemd := IsRunningSystemd()
		assert.Equal(t, isRunningSystemd, true)
		assert.Equal(t, called, true)
	})

	t.Run("returns false when IsSystemd returns false", func(t *testing.T) {
		called := false
		defer gomonkey.ApplyFunc(service.IsSystemd, func() bool {
			called = true
			return false
		}).Reset()
		isRunningSystemd := IsRunningSystemd()
		assert.Equal(t, isRunningSystemd, true)
		assert.Equal(t, called, false)

		resetState() //reset isRunningSystemdOnce
		isRunningSystemd = IsRunningSystemd()
		assert.Equal(t, isRunningSystemd, false)
		assert.Equal(t, called, true)
	})
}
