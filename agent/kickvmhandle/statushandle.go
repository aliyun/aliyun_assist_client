package kickvmhandle

import (
	"errors"

	"github.com/aliyun/aliyun_assist_client/thirdparty/sirupsen/logrus"
	"github.com/spf13/pflag"

	"github.com/aliyun/aliyun_assist_client/agent/checknet"
	"github.com/aliyun/aliyun_assist_client/agent/log"
	"github.com/aliyun/aliyun_assist_client/common/networkcategory"
)

type StatusHandle struct {
	action string
	params []string
}

var (
	statusRoute map[string]handleFunc

	ErrStatusNetworkInvalidParameters = errors.New("Invalid parameters for `kick_vm status network` action")
)

func init() {
	statusRoute = map[string]handleFunc{
		"network": requestNetworkStatus,
	}
}

func NewStatusHandle(action string, params []string) *StatusHandle {
	return &StatusHandle{
		action: action,
		params: params,
	}
}

func (h *StatusHandle) DoAction() error {
	if v, ok := statusRoute[h.action]; ok {
		return v(h.params)
	} else {
		return errors.New("no action found")
	}
}

func (h *StatusHandle) CheckAction() bool {
	if _, ok := statusRoute[h.action]; ok {
		return true
	} else {
		return false
	}
}

func requestNetworkStatus(params []string) error {
	logger := log.GetLogger().WithFields(logrus.Fields{
		"module": "requestNetworkStatus",
	})
	// REMEMBER: All actions in all kick_vm option handlers are not able to
	// return results simultaneously.

	flags := pflag.NewFlagSet("network", pflag.ContinueOnError)
	needToRefresh := flags.Bool("refresh", false, "Request to refresh the network diagnostic result")
	isVPCNetwork := flags.Bool("vpc", false, "Declare the instance running in VPC network")
	isClassicNetwork := flags.Bool("classic", false, "Declare the instance running in classic network")
	// Disable unexpected usage printing when failing to parse kick_vm parameters
	flags.Usage = func() {}

	if err := flags.Parse(params); err != nil {
		logger.WithFields(logrus.Fields{
			"params": params,
		}).WithError(err).Errorln("Failed to parse parameters of `kick_vm status network` action")
		return ErrStatusNetworkInvalidParameters
	}

	// Behave as no-op when `--refresh` option is not specified
	if *needToRefresh == false {
		if *isVPCNetwork == true || *isClassicNetwork == true {
			logger.WithFields(logrus.Fields{
				"params": params,
			}).Errorln("Network category options --vpc and --classic can only be specified when --refresh is specified at first")
			return ErrStatusNetworkInvalidParameters
		}

		return nil
	}

	// Parse other options and request to refresh network diagnostic result
	if *isVPCNetwork == true {
		if *isClassicNetwork == true {
			logger.WithFields(logrus.Fields{
				"params": params,
			}).Errorln("Network category options --vpc and --classic are contradictory")
			return ErrStatusNetworkInvalidParameters
		}

		checknet.DeclareNetworkCategory(networkcategory.NetworkVPC)
	} else if *isClassicNetwork == true {
		if *isVPCNetwork == true {
			logger.WithFields(logrus.Fields{
				"params": params,
			}).Errorln("Network category options --vpc and --classic are contradictory")
			return ErrStatusNetworkInvalidParameters
		}

		checknet.DeclareNetworkCategory(networkcategory.NetworkClassic)
	} else {
		logger.WithFields(logrus.Fields{
			"params": params,
		}).Errorln("One of network category option --vpc or --classic must be specified")
		return ErrStatusNetworkInvalidParameters
	}
	// Actions in kick_vm option handlers are all asynchronously called via new
	// goroutine, thus synchronously calling checknet.RequestNetcheck() here is
	// safe.
	checknet.RequestNetcheck(checknet.NetcheckRequestForceOnce)
	return nil
}
