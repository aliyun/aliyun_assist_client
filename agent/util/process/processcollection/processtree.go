package processcollection

import (
	"os"

	"github.com/aliyun/aliyun_assist_client/agent/log"
)

type ProcessTree struct {
	name        string
	processList []*os.Process
}

func CreateProcessTree(name string) (*ProcessTree, error) {
	return &ProcessTree{
		name: name,
	}, nil
}

func (g *ProcessTree) Name() string {
	return g.name
}

// AddPid add process into group
func (g *ProcessTree) AddProcess(p *os.Process) error {
	log.GetLogger().Infof("ProcessTree add process %d ", p.Pid)
	g.processList = append(g.processList, p)
	return nil
}

// Dispose
func (g *ProcessTree) Dispose() error {
	return nil
}
