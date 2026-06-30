package hybrid

import (
	"sync"

	"github.com/aliyun/aliyun_assist_client/thirdparty/sirupsen/logrus"
	"github.com/tidwall/gjson"

	"github.com/aliyun/aliyun_assist_client/agent/log"
)

var (
	_onInstanceDeregisteredLock sync.Mutex
)

// OnErrorResponse handles the error response from the server, checks error
// message(s) about managed instance, and executes related actions
//
// HTTP status code and error message concerned:
// * [403] instance_deregistered
func OnErrorResponse(statusCode int, content string, err error) {
	// Fast-fail when status code is not concerned
	if statusCode != 403 {
		return
	}

	errMsg := gjson.Get(content, "errMsg")
	if errMsg.Exists() && errMsg.String() == "instance_deregistered" {
		OnInstanceDeregisteredMessage(log.GetLogger().WithFields(logrus.Fields{
			"errMsg": errMsg.String(),
		}))
	}
}

func OnInstanceDeregisteredMessage(logger logrus.FieldLogger) {
	// Use mutex to avoid concurrent cleanup of managed-instance registration
	// record, although it seems to be harmless.
	//
	// Such critical section is unlikely to re-enter, since service process will
	// be stopped right after hybrid.CleanUpRegisterDataAndExit()
	_onInstanceDeregisteredLock.Lock()
	defer _onInstanceDeregisteredLock.Unlock()

	logger.Info("Clean up hybrid instance info and stop agent process self due to errMsg indication")
	CleanUpRegisterDataAndExit()
}
