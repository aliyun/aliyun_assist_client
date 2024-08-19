package processcollection

import "os"

type ProcessCollection interface {
	Name() string
	// AddProcess add process into collection
	AddProcess(*os.Process) error
	// KillAll kill all processes in collection
	KillAll() error
	// Dispose
	Dispose() error
}
