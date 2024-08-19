package cri

type RuntimeEndpoint struct{
	RuntimeName string
	Endpoint string
}

var (
	RuntimeName2Endpoints map[string][]RuntimeEndpoint
)

func init() {
	RuntimeName2Endpoints = make(map[string][]RuntimeEndpoint, len(PrecedentRuntimeEndpoints))
	for _, endpoint := range PrecedentRuntimeEndpoints {
		RuntimeName2Endpoints[endpoint.RuntimeName] = append(RuntimeName2Endpoints[endpoint.RuntimeName], endpoint)
	}
}
