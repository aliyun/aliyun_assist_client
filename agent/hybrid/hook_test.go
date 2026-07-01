package hybrid

import (
	"errors"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/stretchr/testify/assert"
	"github.com/tidwall/gjson"
)

func TestOnErrorResponse(t *testing.T) {
	t.Run("StatusCodeNot403ShouldReturnImmediately", func(t *testing.T) {
		exitCalled := false
		patch := gomonkey.ApplyFunc(CleanUpRegisterDataAndExit, func() {
			exitCalled = true
		})
		defer patch.Reset()

		OnErrorResponse(404, `{"errMsg": "some_error"}`, errors.New("test error"))

		assert.False(t, exitCalled, "CleanUpRegisterDataAndExit should not be called when status code is not 403")
	})

	t.Run("StatusCode403ButErrMsgNotInstanceDeregisteredShouldReturn", func(t *testing.T) {
		exitCalled := false
		patch := gomonkey.ApplyFunc(CleanUpRegisterDataAndExit, func() {
			exitCalled = true
		})
		defer patch.Reset()

		OnErrorResponse(403, `{"errMsg": "other_error"}`, errors.New("test error"))

		assert.False(t, exitCalled, "CleanUpRegisterDataAndExit should not be called when errMsg is not 'instance_deregistered'")
	})

	t.Run("StatusCode403AndErrMsgIsInstanceDeregisteredShouldCallCleanUp", func(t *testing.T) {
		exitCalled := false
		patch := gomonkey.ApplyFunc(CleanUpRegisterDataAndExit, func() {
			exitCalled = true
		})
		defer patch.Reset()

		OnErrorResponse(403, `{"errMsg": "instance_deregistered"}`, errors.New("test error"))

		assert.True(t, exitCalled, "CleanUpRegisterDataAndExit should be called when status code is 403 and errMsg is 'instance_deregistered'")
	})

	t.Run("StatusCode403AndMissingErrMsgShouldReturn", func(t *testing.T) {
		exitCalled := false
		patch := gomonkey.ApplyFunc(CleanUpRegisterDataAndExit, func() {
			exitCalled = true
		})
		defer patch.Reset()

		content := `{"error": "some_other_field"}`
		errMsg := gjson.Get(content, "errMsg")
		assert.False(t, errMsg.Exists(), "errMsg should not exist in content")

		OnErrorResponse(403, content, errors.New("test error"))

		assert.False(t, exitCalled, "CleanUpRegisterDataAndExit should not be called when errMsg does not exist")
	})

	t.Run("StatusCode403AndEmptyContentShouldReturn", func(t *testing.T) {
		exitCalled := false
		patch := gomonkey.ApplyFunc(CleanUpRegisterDataAndExit, func() {
			exitCalled = true
		})
		defer patch.Reset()

		OnErrorResponse(403, `{}`, errors.New("test error"))

		assert.False(t, exitCalled, "CleanUpRegisterDataAndExit should not be called when content is empty")
	})
}
