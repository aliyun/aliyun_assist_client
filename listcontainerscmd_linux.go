//go:build linux

package main

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/aliyun/aliyun_assist_client/agent/log"
	"github.com/aliyun/aliyun_assist_client/agent/commandermanager"
	"github.com/aliyun/aliyun_assist_client/thirdparty/aliyun-cli/cli"
	"github.com/aliyun/aliyun_assist_client/thirdparty/aliyun-cli/i18n"
)

const (
	listContainersCmdName = "list-containers"
)

var (
	listContainersCmd = &cli.Command{
		Name:               listContainersCmdName,
		Short:              i18n.T("List containers on the instance", "列出该实例上的容器"),
		Usage:              fmt.Sprintf("%s [flags]", listContainersCmdName),
		Sample:             "",
		DisableFlagParsing: true,
		Run:                runListContainersCmd,
	}
)

func runListContainersCmd(ctx *cli.Context, args []string) error {
	// Extract value of persistent flags
	logPath, _ := ctx.Flags().Get(LogPathFlagName).GetValue()
	log.InitLog("aliyun_assist_main.log", logPath, true)
	log.GetLogger().Info("Sub command start: ", listContainersCmdName)
	commandermanager.InitCommanderManager(commandermanager.ContainerCommanderName)
	containercommander, err := commandermanager.GetCommander(commandermanager.ContainerCommanderName)
	if err != nil {
		fmt.Println(err)
		cli.Exit(1)
	}

	pluginPath := containercommander.CmdPath()
	log.GetLogger().Infof("Plugin[%s]: %s",commandermanager.ContainerCommanderName, pluginPath)
	log.GetLogger().Info("Args: ", args)

	args = append([]string{listContainersCmdName}, args...)
	cmd := exec.Command(pluginPath, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Println(err)
		log.GetLogger().Errorf("Run plugin[%s] failed: %v", commandermanager.ContainerCommanderName, err)
		cli.Exit(cmd.ProcessState.ExitCode())
	}
	return nil
}
