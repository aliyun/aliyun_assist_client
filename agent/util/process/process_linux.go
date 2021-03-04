package process

import (
	"syscall"

	"github.com/aliyun/aliyun_assist_client/agent/log"
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
	log.GetLogger().Infoln("addCredential")
	uid, gid, groups, err := GetUserCredentials(p.user_name)
	if err != nil {
		return err
	}

	if p.command.SysProcAttr == nil {
		p.command.SysProcAttr = &syscall.SysProcAttr{}
	}
	p.command.SysProcAttr.Credential = &syscall.Credential{Uid: uid, Gid: gid, Groups: groups, NoSetGroups: false}

	// Setting home environment variable for RunAs user
	runAsUserHomeEnvVariable := "HOME=/home/" + p.user_name
	p.command.Env = append(p.command.Env, runAsUserHomeEnvVariable)

	return nil
}

func (p *ProcessCmd)  removeCredential () error {
	return nil
}

func IsUserValid (userName string, password string) error {
	return nil
}

