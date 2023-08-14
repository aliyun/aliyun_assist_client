// +build linux freebsd

package process

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"

	"github.com/aliyun/aliyun_assist_client/agent/log"
)
const sudoersFileReadWriteMode = 0640
const sudoersFileReadOnlyMode = 0440

var ShellPluginCommandName = "sh"
var ShellPluginCommandArgs = []string{"-c"}

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
		// append specific envs to osEnv, the value of repetitive key will be covered
		env = os.Environ()
		for i:=0; i<len(p.command.Env); i++ {
			env = append(env, p.command.Env[i])
		}
	}
	// 2. Append correct $HOME environment variable value
	if p.homeDir != "" {
		homeEnv := fmt.Sprintf("HOME=%s", p.homeDir)
		env = append(env, homeEnv)
	}

	p.command.Env = env

	return nil
}

func DoesUserExist(username string) (bool, error) {
	shellCmdArgs := append(ShellPluginCommandArgs, fmt.Sprintf("id %s", username))
	cmd := exec.Command(ShellPluginCommandName, shellCmdArgs...)
	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			// The program has exited with an exit code != 0
			return false, fmt.Errorf("encountered an error while checking for %s: %v", username, exitErr.Error())
		}
		return false, nil
	}
	return true, nil
}

func CreateLocalAdminUser(defaultRunAsUserName string) (err error) {
    err = nil
	userExists, _ := DoesUserExist(defaultRunAsUserName)

	if userExists {
		log.GetLogger().Infof("%s already exists.", defaultRunAsUserName)
	} else {
		if err = createLocalUser(defaultRunAsUserName); err != nil {
			return err
		}
		// only create sudoers file when user does not exist
		err = createSudoersFileIfNotPresent(defaultRunAsUserName)
	}

	return err
}

func  createLocalUser(defaultRunAsUserName string) error {

	commandArgs := append(ShellPluginCommandArgs, fmt.Sprintf(createUserCommandFormater, defaultRunAsUserName))
	cmd := exec.Command(ShellPluginCommandName, commandArgs...)
	if err := cmd.Run(); err != nil {
		log.GetLogger().Errorf("Failed to create %s: %v", defaultRunAsUserName, err)
		return err
	}
	log.GetLogger().Infof("Successfully created %s", defaultRunAsUserName)
	return nil
}

// createSudoersFileIfNotPresent will create the sudoers file if not present.
func  createSudoersFileIfNotPresent(defaultRunAsUserName string) error {

	// Return if the file exists
	if _, err := os.Stat(sudoersFile); err == nil {
		log.GetLogger().Infof("File %s already exists", sudoersFile)
		changeModeOfSudoersFile()
		return err
	}

	// Create a sudoers file for ecs-assist-user
	file, err := os.OpenFile(sudoersFile, os.O_WRONLY|os.O_CREATE, sudoersFileReadWriteMode)
	if err != nil {
		log.GetLogger().Errorf("Failed to add %s to sudoers file: %v", defaultRunAsUserName, err)
		return err
	}
	defer file.Close()

	file.WriteString(fmt.Sprintf("# User rules for %s\n", defaultRunAsUserName))
	file.WriteString(fmt.Sprintf("%s ALL=(ALL) NOPASSWD:ALL\n", defaultRunAsUserName))
	log.GetLogger().Infof("Successfully created file %s", sudoersFile)
	changeModeOfSudoersFile()
	return nil
}

// changeModeOfSudoersFile will change the sudoersFile mode to 0440 (read only).
// This file is created with mode 0640 using os.OpenFile() so needs to be updated to read only with chmod.
func  changeModeOfSudoersFile() error {
	fileMode := os.FileMode(sudoersFileReadOnlyMode)
	if err := os.Chmod(sudoersFile, fileMode); err != nil {
		log.GetLogger().Errorf("Failed to change mode of %s to %d: %v", sudoersFile, sudoersFileReadOnlyMode, err)
		return err
	}
	log.GetLogger().Infof("Successfully changed mode of %s to %d", sudoersFile, sudoersFileReadOnlyMode)
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

	return nil
}

func (p *ProcessCmd)  removeCredential () error {
	return nil
}

func IsUserValid (userName string, password string) error {
	return nil
}
