package buses

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
		path:     path,
	}
}

func (e *Endpoint) GetProtocol() string { return e.protocol }
func (e *Endpoint) GetPath() string     { return e.path }
