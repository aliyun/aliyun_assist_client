package heartbeat

import (
	"errors"
	"regexp"

	"github.com/aliyun/aliyun_assist_client/agent/log"
	"github.com/aliyun/aliyun_assist_client/thirdparty/sirupsen/logrus"
	"github.com/tidwall/gjson"
)

type ErrorMessageHandler func(logger logrus.FieldLogger)

var (
	errMissingRequiredField = errors.New("request field missing")

	// _fieldMissRegexp is used to match error messages for missing fields,
	// like: "Required request parameter 'os_type' for method parameter type String is not present"
	_fieldMissRegexp = regexp.MustCompile(`Required request parameter '(\w+)' for method parameter type (\w+) is not present`)

	_instanceDeregisteredHook ErrorMessageHandler
)

func RegisterInstanceDeregisteredHook(handler ErrorMessageHandler) {
	_instanceDeregisteredHook = handler
}

func digErrorFromResponse(responseContent string) error {
	errMsg := extractErrMsg(responseContent)
	if errMsg == "" {
		return nil
	}

	logger := log.GetLogger().WithField("errMsg", errMsg)
	logger.Warning("Error message presented in heart-beat response")
	if miss, fieldName, fieldType := extractMissingFieldFromMessage(errMsg); miss {
		_useFullFields.Store(true)
		logger.WithFields(logrus.Fields{
			"fieldName": fieldName,
			"fieldType": fieldType,
		}).Error("Missing required field in current heart-beat request")
		return errMissingRequiredField
	} else if errMsg == "instance_deregistered" {
		if _instanceDeregisteredHook != nil {
			defer _instanceDeregisteredHook(logger)
		}
	}

	return nil
}

func extractErrMsg(content string) string {
	if !gjson.Valid(content) {
		log.GetLogger().WithFields(log.Fields{
			"response": content,
		}).Errorln("Invalid json response")
		return ""
	}

	json := gjson.Parse(content)
	errMsgField := json.Get("errMsg")
	if !errMsgField.Exists() {
		return ""
	}

	errMsg, ok := errMsgField.Value().(string)
	if !ok {
		log.GetLogger().WithFields(log.Fields{
			"response": content,
		}).Errorln("Invalid errMsg value in json response")
		return ""
	}

	return errMsg
}

func extractMissingFieldFromMessage(errMsg string) (matched bool, fieldName string, fieldType string) {
	if _fieldMissRegexp.MatchString(errMsg) {
		matched = true
		items := _fieldMissRegexp.FindStringSubmatch(errMsg)
		if len(items) != 3 {
			return
		}
		fieldName = items[1]
		fieldType = items[2]
	}
	return
}
