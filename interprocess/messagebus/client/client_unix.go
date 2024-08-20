//go:build darwin || freebsd || linux
// +build darwin freebsd linux

package client

import (
	"context"
	"net"

	"github.com/aliyun/aliyun_assist_client/interprocess/messagebus/buses"
)

func getContextDialerForEndpoint(endpoint buses.Endpoint) (func (ctx context.Context, addr string) (net.Conn, error), error) {
	switch endpoint.GetProtocol() {
	case buses.UnixDomainSocketProtocol:
		return dialUDSEndpoint, nil
	default:
		return nil, ErrUnsupportedEndpointProtocol
	}
}

func dialUDSEndpoint(ctx context.Context, addr string) (net.Conn, error) {
	return (&net.Dialer{}).DialContext(ctx, buses.UnixDomainSocketProtocol, addr)
}
