package model

type DataSourceName string
const (
	ViaCRI DataSourceName = "CRI"
	ViaDocker DataSourceName = "docker"
)

type Container struct {
	Id string `json:"id"`
	Name string `json:"name"`
	PodId string `json:"podId,omitempty"`
	PodName string `json:"podName,omitempty"`
	RuntimeName string `json:"runtimeName,omitempty"`
	State string `json:"state"`
	DataSource DataSourceName `json:"dataSource"`
}
