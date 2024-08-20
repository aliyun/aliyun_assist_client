package client

import (
	"context"
	"net"

	"github.com/Microsoft/go-winio"

	"github.com/aliyun/aliyun_assist_client/interprocess/messagebus/buses"
)

func getContextDialerForEndpoint(endpoint buses.Endpoint) (func (ctx context.Context, addr string) (net.Conn, error), error) {
	switch endpoint.GetProtocol() {
	case buses.NamedPipeProtocol:
		return winio.DialPipeContext, nil
	default:
		return nil, ErrUnsupportedEndpointProtocol
	}
}
