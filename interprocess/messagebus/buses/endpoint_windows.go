package buses

import (
	"fmt"
)

const (
	centralNamedPipeName = "aliyun_assist_service.ipc"
)

func GetCentralEndpoint(overWrite bool) Endpoint {
	npipePath := fmt.Sprintf(`\\.\pipe\%s`, centralNamedPipeName)
	return Endpoint{
		protocol: NamedPipeProtocol,
		path: npipePath,
	}
}
