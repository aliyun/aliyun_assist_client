package acspluginmanager

const (
	FetchFromLocalFile      = "LocalFile"
	FetchFromLocalInstalled = "LocalInstalled"
	FetchFromOnline         = "Online"
)

type Fetched struct {
	PluginName    string
	PluginVersion string
	PluginType    string

	Entrypoint                string
	ExecutionTimeoutInSeconds int
	EnvPluginDir              string
	EnvPrePluginDir           string
}
