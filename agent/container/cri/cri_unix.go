//go:build !windows
// +build !windows

package cri

var (
	PrecedentRuntimeEndpoints = []RuntimeEndpoint{
		{"docker", "unix:///var/run/dockershim.sock"},
		{"containerd", "unix:///run/containerd/containerd.sock"},
		{"cri-o", "unix:///run/crio/crio.sock"},
		{"docker", "unix:///var/run/cri-dockerd.sock"},
	}
)
