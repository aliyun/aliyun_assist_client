package process

import "syscall"

import (
	"github.com/aliyun/aliyun_assist_client/agent/log"
)

func (p *ProcessCmd)  addCredential () error {
	log.GetLogger().Infoln("addCredential")
	uid, gid, groups, err := GetUserCredentials(p.user_name)
	if err != nil {
		return err
	}
	p.command.SysProcAttr = &syscall.SysProcAttr{}
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

