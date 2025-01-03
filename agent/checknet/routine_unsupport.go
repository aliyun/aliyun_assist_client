//go:build freebsd

package checknet

import (
	"errors"
	"github.com/aliyun/aliyun_assist_client/common/networkcategory"
	"github.com/aliyun/aliyun_assist_client/thirdparty/sirupsen/logrus"
)

var (
	ErrNotSupported = errors.New("Not supported")
)

// RequestNetcheck is currently not supported on this operating system, and is
// simply an empty function. On supported OSes, it would asynchronously invoke
// netcheck program for network diagnostic, when no other network diagnostic is
// running or the last diagnostic report has outdated.
func RequestNetcheck(requestType NetcheckRequestType) {
}

// RecentReport is currently not supported on this operating system, and simply
// returns nil pointer. On supported OSes, it would return the most recent
// available network diagnostic report, or nil pointer if the report has not
// been generated.
func RecentReport() *CheckReport {
	return nil
}

// DeclareNetworkCategory is currently not supported on this operating system,
// and is simply an empty function. On supported OSes, it would set the network
// category in cache of this module, which is used to specify the network
// environment when running netcheck program.
func DeclareNetworkCategory(category networkcategory.NetworkCategory) {
}

// CollectNetworkConfiguration is currently not supported on this operating system,
// and is simply an empty function. On supported OSes, it would collect the
// network configurations of system, which be used to diagnosing no network issues.
func CollectNetworkConfiguration(logger logrus.FieldLogger, taskId string) (int, error) {
	return 0, ErrNotSupported
}
