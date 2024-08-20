package buses

import (
	"fmt"
	"strings"
)

const (
	NamedPipeProtocol        = "npipe"
	UnixDomainSocketProtocol = "unix"
)

type Endpoint struct {
	protocol string
	path     string
}

func NewEndpoint(protocol, path string) Endpoint {
	return Endpoint{
		protocol: protocol,
		path: path,
	}
}

func (e *Endpoint) Parse(endpoint string) error {
	items := strings.Split(endpoint, "://")
	if len(items) != 2 {
		return fmt.Errorf("unknown endpoint")
	}
	e.protocol = items[0]
	e.path = items[1]
	return nil
}
func (e *Endpoint) GetProtocol() string { return e.protocol }
func (e *Endpoint) GetPath() string { return e.path }
func (e *Endpoint) String() string { return fmt.Sprintf("%s://%s", e.protocol, e.path)}
