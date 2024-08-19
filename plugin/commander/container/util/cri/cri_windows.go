package cri

var (
	PrecedentRuntimeEndpoints = []RuntimeEndpoint{
		{"docker", "npipe:////./pipe/dockershim"},
		{"containerd", "npipe:////./pipe/containerd-containerd"},
		{"docker", "npipe:////./pipe/cri-dockerd"},
	}
)
