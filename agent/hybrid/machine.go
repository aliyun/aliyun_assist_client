package hybrid

type MachineIdentifier interface {
	Name() string

	Generate() (string, error)
}

type MachineIdentifyCleaner interface {
	MachineIdentifier

	Cleanup() error
}

type MachineIdentifyRefresher interface {
	MachineIdentifier

	Refresh() (string, error)
}
