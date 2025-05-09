package pluginmodel

type RemotePlugin interface {
	Name() string

	Version() string

	TimeoutSecs() int

	Url() string
}
