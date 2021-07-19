package process

import (
	"fmt"
	"os"
	"syscall"
)

func (p *ProcessCmd) prepareProcess() error {
	if p.command.SysProcAttr == nil {
		p.command.SysProcAttr = &syscall.SysProcAttr{
			Setpgid: true,
			Pgid: 0,
		}
	}

	// Fix environment variable $HOME
	var env []string
	// 1. Duplicate current environment variable settings as base
	if p.command.Env == nil || len(p.command.Env) == 0 {
		env = os.Environ()
	} else {
		env = p.command.Env
	}
	// 2. Append correct $HOME environment variable value
	if p.homeDir != "" {
		homeEnv := fmt.Sprintf("HOME=%s", p.homeDir)
		env = append(env, homeEnv)
	}

	p.command.Env = env

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