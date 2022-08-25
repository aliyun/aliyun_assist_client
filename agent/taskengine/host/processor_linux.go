package host

import (
	"fmt"

	"github.com/aliyun/aliyun_assist_client/agent/taskengine/taskerrors"
	"github.com/aliyun/aliyun_assist_client/agent/util/process"
)

func (p *HostProcessor) checkCredentials() (bool, error) {
	if _, _, _, err := process.GetUserCredentials(p.Username); err != nil {
		return false, taskerrors.NewInvalidUsernameOrPasswordError(err, fmt.Sprintf("UserInvalid_%s", p.Username))
	}

	return true, nil
}
