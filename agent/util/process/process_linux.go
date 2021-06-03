package process

import (
	"os"
	"os/exec"
	"syscall"
     "fmt"
	"github.com/aliyun/aliyun_assist_client/agent/log"
)

const sudoersFile = "/etc/sudoers.d/ecs-assist-user"
const sudoersFileMode = 0440

var ShellPluginCommandName = "sh"
var ShellPluginCommandArgs = []string{"-c"}

func (p *ProcessCmd) prepareProcess() error {
	if p.command.SysProcAttr == nil {
		p.command.SysProcAttr = &syscall.SysProcAttr{
			Setpgid: true,
			Pgid: 0,
		}
	}

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

	commandArgs := append(ShellPluginCommandArgs, fmt.Sprintf("useradd -m %s -s /sbin/nologin", defaultRunAsUserName))
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

	// Create a sudoers file for ssm-user
	file, err := os.Create(sudoersFile)
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
// This file is created with mode 0666 using os.Create() so needs to be updated to read only with chmod.
func  changeModeOfSudoersFile() error {
	fileMode := os.FileMode(sudoersFileMode)
	if err := os.Chmod(sudoersFile, fileMode); err != nil {
		log.GetLogger().Errorf("Failed to change mode of %s to %d: %v", sudoersFile, sudoersFileMode, err)
		return err
	}
	log.GetLogger().Infof("Successfully changed mode of %s to %d", sudoersFile, sudoersFileMode)
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

