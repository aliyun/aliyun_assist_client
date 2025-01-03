package buses

import (
	"fmt"
	"strings"
)

const (
	centralNamedPipeName = "aliyun_assist_service.ipc"
)

// Unix domain sockets (UDS) is a widely supported IPC technology. UDS is the best choice for building cross-platform apps, and it's usable on Linux, macOS, and Windows 10/Windows Server 2019 or later.
// Named pipes are supported by all versions of Windows. Named pipes integrate well with Windows security, which can control client access to the pipe.
func GetCentralEndpoint(overWrite bool) Endpoint {
	npipePath := fmt.Sprintf(`\\.\pipe\%s`, centralNamedPipeName)
	return Endpoint{
		protocol: NamedPipeProtocol,
		path:     npipePath,
	}
}

func (e *Endpoint) Parse(endpoint string) error {
	items := strings.Split(endpoint, "://")
	if len(items) != 2 {
		return fmt.Errorf("unknown endpoint")
	}
	e.protocol = items[0]
	if e.protocol == NamedPipeProtocol {
		e.path = fmt.Sprintf(`\\.\pipe\%s`, items[1])
	} else {
		e.path = items[1]
	}
	return nil
}

func (e *Endpoint) String() string { 
	return fmt.Sprintf("%s://%s", e.protocol, strings.TrimPrefix(e.path, `\\.\pipe\`)) 
}
