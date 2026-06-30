package process

import (
	"errors"
	"fmt"
	"os"
	"syscall"

	"github.com/aliyun/aliyun_assist_client/agent/cryptdata"
	"github.com/aliyun/aliyun_assist_client/agent/log"
	"github.com/aliyun/aliyun_assist_client/agent/util/paramstore"
	"github.com/aliyun/aliyun_assist_client/agent/util/winapi"
	"github.com/aliyun/aliyun_assist_client/thirdparty/sirupsen/logrus"
	"golang.org/x/sys/windows"
)

func (p *ProcessCmd) prepareProcess() error {
	// if p.command.SysProcAttr == nil {
	// 	p.command.SysProcAttr = &windows.SysProcAttr{}
	// }
	// 1. Duplicate current environment variable settings as base
	var env []string
	if p.command.Env == nil || len(p.command.Env) == 0 {
		env = os.Environ()
	} else {
		// append specific envs to osEnv, the value of repetitive key will be covered
		env = os.Environ()
		for i := 0; i < len(p.command.Env); i++ {
			env = append(env, p.command.Env[i])
		}
	}
	p.command.Env = env

	for _, opt := range p.commandOptions {
		if err := opt(p.command); err != nil {
			return err
		}
	}

	return nil
}

func (p *ProcessCmd) addCredential() error {
	logger := log.GetLogger()
	logger.Infoln("addCredential")
	vm_password, err := GetSecretParam(logger, p.password)
	if err != nil {
		logger.Errorln("get password failed", err)
		return err
	}
	token, err := winapi.LogonUser(logger, p.user_name, vm_password, winapi.Logon32LogonInteractive)
	if err != nil {
		return err
	}

	p.command.SysProcAttr = &windows.SysProcAttr{
		Token: syscall.Token(token),
	}

	return nil
}

func (p *ProcessCmd) removeCredential() error {
	p.command.SysProcAttr.Token.Close()

	return nil
}

func IsUserValid(userName string, password string) error {
	logger := log.GetLogger()
	vm_password, err := GetSecretParam(logger, password)
	if err != nil {
		return err
	}
	token, err := winapi.LogonUser(logger, userName, vm_password, winapi.Logon32LogonInteractive)
	if err != nil {
		logger.WithError(err).Errorf("Authentication failed for user %s with password", userName)
		return errors.New("UsernameOrPasswordInvalid")
	}
	defer winapi.MustCloseHandle(logger, token)
	return nil
}

func GetSecretParam(logger logrus.FieldLogger, secretName string) (string, error) {
	var value string
	var paramValueInfo *cryptdata.ParamValueInfo
	var err_1, err_2 error
	if paramValueInfo, err_1 = cryptdata.GetSecretParamValue(secretName); err_1 != nil {
		if value, err_2 = paramstore.GetSecretParam(secretName); err_2 != nil {
			logger.Errorf("Secret param '%s' not found in agent [%v] and oos[%v]", secretName, err_1, err_2)
			err := fmt.Errorf("Secret param '%s' not found in agent [%v] and oos[%v]", secretName, err_1, err_2)
			return "", err
		}
	} else {
		value = paramValueInfo.SecretValue
	}

	return value, nil
}
