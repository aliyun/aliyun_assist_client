package heartbeat

import (
	"testing"

	"github.com/aliyun/aliyun_assist_client/thirdparty/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestDigErrorFromResponse(t *testing.T) {
	t.Run("ValidJsonWithNoErrMsgShouldReturnNil", func(t *testing.T) {
		responseContent := `{"status": "success"}`
		err := digErrorFromResponse(responseContent)
		assert.NoError(t, err)
	})

	t.Run("InvalidJsonShouldReturnNil", func(t *testing.T) {
		responseContent := `invalid json`
		err := digErrorFromResponse(responseContent)
		assert.NoError(t, err)
	})

	t.Run("MissingRequiredFieldErrorMessageShouldSetUseFullFields", func(t *testing.T) {
		responseContent := `{"errMsg": "Required request parameter 'os_type' for method parameter type String is not present"}`
		// Store original value
		originalValue := _useFullFields.Load()
		defer _useFullFields.Store(originalValue)
		err := digErrorFromResponse(responseContent)
		assert.ErrorIs(t, err, errMissingRequiredField)
		// Check if _useFullFields is set to true
		assert.True(t, _useFullFields.Load())
	})

	t.Run("InstanceDeregisteredErrorMessageShouldCallHook", func(t *testing.T) {
		responseContent := `{"errMsg": "instance_deregistered"}`
		hookCalled := false

		RegisterInstanceDeregisteredHook(func(logger logrus.FieldLogger) {
			hookCalled = true
		})

		err := digErrorFromResponse(responseContent)
		assert.NoError(t, err)
		assert.True(t, hookCalled)
	})

	t.Run("OtherErrorMessageShouldReturnNil", func(t *testing.T) {
		responseContent := `{"errMsg": "some_other_error"}`
		err := digErrorFromResponse(responseContent)
		assert.NoError(t, err)
	})
}

func TestExtractErrMsg(t *testing.T) {
	t.Run("ValidJsonWithErrMsgShouldReturnMessage", func(t *testing.T) {
		content := `{"errMsg": "test error message"}`
		result := extractErrMsg(content)
		assert.Equal(t, "test error message", result)
	})

	t.Run("ValidJsonWithoutErrMsgShouldReturnEmpty", func(t *testing.T) {
		content := `{"status": "success"}`
		result := extractErrMsg(content)
		assert.Empty(t, result)
	})

	t.Run("InvalidJsonShouldReturnEmpty", func(t *testing.T) {
		content := `invalid json content`
		result := extractErrMsg(content)
		assert.Empty(t, result)
	})

	t.Run("ErrMsgNotStringShouldReturnEmpty", func(t *testing.T) {
		content := `{"errMsg": 123}`
		result := extractErrMsg(content)
		assert.Empty(t, result)
	})
}

func TestExtractMissingFieldFromMessage(t *testing.T) {
	t.Run("NormalCase", func(t *testing.T) {
		errMsg := "Required request parameter 'os_type' for method parameter type String is not present"
		matched, fieldName, fieldType := extractMissingFieldFromMessage(errMsg)
		assert.True(t, matched)
		assert.Equal(t, "os_type", fieldName)
		assert.Equal(t, "String", fieldType)
	})

	t.Run("NonMatchingMessage", func(t *testing.T) {
		errMsg := "Some other error message"
		matched, fieldName, fieldType := extractMissingFieldFromMessage(errMsg)
		assert.False(t, matched)
		assert.Empty(t, fieldName)
		assert.Empty(t, fieldType)
	})

	t.Run("MatchingMessageWithDifferentField", func(t *testing.T) {
		errMsg := "Required request parameter 'instance_id' for method parameter type Integer is not present"
		matched, fieldName, fieldType := extractMissingFieldFromMessage(errMsg)
		assert.True(t, matched)
		assert.Equal(t, "instance_id", fieldName)
		assert.Equal(t, "Integer", fieldType)
	})
}
