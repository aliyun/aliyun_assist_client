package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/aliyun/aliyun_assist_client/agent/log"
	pb "github.com/aliyun/aliyun_assist_client/interprocess/configure/agrpc"
	"github.com/aliyun/aliyun_assist_client/interprocess/configure/client"
	"github.com/aliyun/aliyun_assist_client/interprocess/configure/server"
	"github.com/aliyun/aliyun_assist_client/thirdparty/aliyun-cli/cli"
	"github.com/aliyun/aliyun_assist_client/thirdparty/aliyun-cli/i18n"
)

const (
	ConfigureSubCmd = "configure"

	getSubSubCmd    = "get"
	setSubSubCmd    = "set"
	reloadSubSubCmd = "reload"

	runtimeFlagName      = "runtime"
	crossVersionFlagName = "cross-version"
	itemFlagName         = "item"
)

var (
	runtimeFlag = cli.Flag{
		Name: runtimeFlagName,
		Short: i18n.T(
			`For runtime only.`,
			`仅对当前运行时生效.`,
		),
		AssignedMode: cli.AssignedNone,
		Category:     "caller",
	}
	crossVersionFlag = cli.Flag{
		Name: crossVersionFlagName,
		Short: i18n.T(
			`Effective across versions.`,
			`跨版本范围生效.`,
		),
		AssignedMode: cli.AssignedNone,
		Category:     "caller",
	}
	itemFlag = cli.Flag{
		Name: itemFlagName,
		Short: i18n.T(
			`Specify configuration in the key=value format.`,
			`按照 key=value 的格式指定配置`,
		),
		AssignedMode: cli.AssignedRepeatable,
		Category:     "caller",
	}

	configureSubCmd = &cli.Command{
		Name: ConfigureSubCmd,
		Short: i18n.T(
			"Manage configuration items",
			"管理配置项",
		),
		EnableUnknownFlag: false,
	}

	getConfigSubSubCmd = &cli.Command{
		Name: getSubSubCmd,
		Short: i18n.T(
			"Get configurations.",
			"查看配置.",
		),
		Usage:             fmt.Sprint(getSubSubCmd, " [flags]"),
		Sample:            getConfigSample(),
		EnableUnknownFlag: false,
		Run:               getConfig,
	}

	setConfigSubSubCmd = &cli.Command{
		Name: setSubSubCmd,
		Short: i18n.T(
			"Set configurations.",
			"修改配置.",
		),
		Usage:             fmt.Sprint(setSubSubCmd, " [flags]"),
		Sample:            setConfigSample(),
		EnableUnknownFlag: false,
		Run:               setConfig,
	}

	reloadConfigSubSubCmd = &cli.Command{
		Name: reloadSubSubCmd,
		Short: i18n.T(
			"Reload configurations.",
			"重新加载配置.",
		),
		Usage:             fmt.Sprint(reloadSubSubCmd),
		EnableUnknownFlag: false,
		Run:               reloadConfig,
	}
)

func init() {
	getConfigSubSubCmd.Flags().Add(&runtimeFlag)
	setConfigSubSubCmd.Flags().Add(&runtimeFlag)
	setConfigSubSubCmd.Flags().Add(&crossVersionFlag)
	setConfigSubSubCmd.Flags().Add(&itemFlag)

	configureSubCmd.AddSubCommand(getConfigSubSubCmd)
	configureSubCmd.AddSubCommand(setConfigSubSubCmd)
	configureSubCmd.AddSubCommand(reloadConfigSubSubCmd)
}

func getConfig(ctx *cli.Context, args []string) error {
	runtime := ctx.Flags().Get(runtimeFlagName).IsAssigned()
	var err error
	var items []*pb.ConfItem
	if runtime {
		items, err = client.GetConf(runtime)
	} else {
		items = server.GetConfLocal(log.GetLogger())
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		cli.Exit(1)
	}
	for _, item := range items {
		fmt.Printf("%s=%s\n", item.Name, item.Value)
	}
	return nil
}

func setConfig(ctx *cli.Context, args []string) error {
	runtime := ctx.Flags().Get(runtimeFlagName).IsAssigned()
	crossVer := ctx.Flags().Get(crossVersionFlagName).IsAssigned()
	items := ctx.Flags().Get(itemFlagName).GetValues()
	if runtime && crossVer {
		fmt.Fprintf(os.Stderr, "`--%s` and `--%s` are in conflict\n", runtimeFlagName, crossVersionFlagName)
		cli.Exit(1)
	}

	var kvList []string
	for _, item := range items {
		idx := strings.Index(item, "=")
		if idx == -1 || idx == len(item)-1 {
			fmt.Fprintf(os.Stderr, "invalid item `%s`\n", item)
			cli.Exit(1)
		}
		kvList = append(kvList, item[:idx])
		kvList = append(kvList, item[idx+1:])
	}
	var err error
	if runtime {
		err = client.SetConf(kvList, runtime, crossVer)
	} else {
		err = server.SetConfLocal(log.GetLogger(), kvList, crossVer)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
	}
	cli.Exit(1)
	
	return nil
}

func reloadConfig(ctx *cli.Context, args []string) error {
	if err := client.ReloadConf(); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		cli.Exit(1)
	}
	return nil
}

func getConfigSample() string {
	s := ""
	path, _ := os.Executable()
	_, exeFile := filepath.Split(path)
	s += fmt.Sprintf("%s %s %s [--%s]", exeFile, ConfigureSubCmd, getSubSubCmd, runtimeFlagName)
	return s
}

func setConfigSample() string {
	s := ""
	path, _ := os.Executable()
	_, exeFile := filepath.Split(path)
	s += fmt.Sprintf("%s %s %s --%s key=value", exeFile, ConfigureSubCmd, setSubSubCmd, itemFlagName)
	s += fmt.Sprintf("\n  %s %s %s --%s --%s key=value", exeFile, ConfigureSubCmd, setSubSubCmd, runtimeFlagName, itemFlagName)
	s += fmt.Sprintf("\n  %s %s %s --%s --%s key=value", exeFile, ConfigureSubCmd, setSubSubCmd, crossVersionFlagName, itemFlagName)
	return s
}
