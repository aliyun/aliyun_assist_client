package acspluginmanager

const (
	FetchFromLocalFile      = "LocalFile"
	FetchFromLocalInstalled = "LocalInstalled"
	FetchFromOnline         = "Online"

	AddSysTag    = "add"
	RemoveSysTag = "remove"
)

type Fetched struct {
	PluginName    string
	PluginVersion string
	PluginType    string
	AddSysTag     bool

	Entrypoint                string
	ExecutionTimeoutInSeconds int
	EnvPluginDir              string
	EnvPrePluginDir           string
}
