package process

import (
	"syscall"
)

func (p *ProcessCmd) prepareProcess() error {
	if p.command.SysProcAttr == nil {
		p.command.SysProcAttr = &syscall.SysProcAttr{
			Setpgid: true,
			Pgid: 0,
		}
	}

	return nil
}

func (p *ProcessCmd)  addCredential () error {
	return nil
}

func IsUserValid (userName string, password string) error {
	return nil
}

func (p *ProcessCmd)  removeCredential () error {
	return nil
}