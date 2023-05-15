//go:build !linux
// +build !linux

package checknet

import (
	"github.com/aliyun/aliyun_assist_client/agent/util/networkcategory"
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
