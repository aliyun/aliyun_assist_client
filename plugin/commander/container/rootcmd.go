package main

import (
	"fmt"

	"github.com/aliyun/aliyun_assist_client/thirdparty/aliyun-cli/cli"
	"github.com/aliyun/aliyun_assist_client/thirdparty/aliyun-cli/i18n"
	"github.com/aliyun/aliyun_assist_client/agent/log"
)

type Options struct {
	GetHelp    bool
	GetVersion bool
	GetGitHash bool

	LogPath string
	IsVerbose bool
}

const (
	HelpFlagName    = "help"
	VersionFlagName = "version"
	GithashFlagName = "githash"
	LogPathFlagName = "logPath"
)

var (
	persistentFlags = []cli.Flag{
		{
			Name:         HelpFlagName,
			Shorthand:    'h',
			Short:        i18n.T(`print help`, `打印此帮助`),
			AssignedMode: cli.AssignedNone,
			Persistent:   true,
			Category:     "caller",
		},
		{
			Name:         LogPathFlagName,
			Shorthand:    'L',
			Short:        i18n.T(`set log dir`, `设置日志目录`),
			AssignedMode: cli.AssignedOnce,
			Persistent:   true,
			Category:     "caller",
		},
	}
	rootFlags = []cli.Flag{
		{
			Name:         VersionFlagName,
			Shorthand:    'v',
			Short:        i18n.T(`print version`, `打印版本号`),
			AssignedMode: cli.AssignedNone,
			Category:     "caller",
		},
		{
			Name:         GithashFlagName,
			Short:        i18n.T(`print git hash`, `打印Git commit哈希值`),
			AssignedMode: cli.AssignedNone,
			Category:     "caller",
		},
	}

	rootCmd = cli.Command{
		Name:              "container-commander",
		Short:             i18n.T(`Aliyun Assist Copyright (c) 2017-2023 Alibaba Group Holding Limited`, `Aliyun Assist Copyright (c) 2017-2023 Alibaba Group Holding Limited`),
		Usage:             "container-commander [subcommand] [flags]",
		Sample:            "",
		EnableUnknownFlag: false,
		Run:               runRootCommand,
	}
)

func init() {
	for i := range persistentFlags {
		rootCmd.Flags().Add(&persistentFlags[i])
	}
	for j := range rootFlags {
		rootCmd.Flags().Add(&rootFlags[j])
	}
}

func parseOptions(ctx *cli.Context) Options {
	options := Options{}

	options.GetHelp = ctx.Flags().Get(HelpFlagName).IsAssigned()
	options.GetVersion = ctx.Flags().Get(VersionFlagName).IsAssigned()
	options.GetGitHash = ctx.Flags().Get(GithashFlagName).IsAssigned()
	options.LogPath, _ = ctx.Flags().Get(LogPathFlagName).GetValue()
	
	return options
}

func runRootCommand(ctx *cli.Context, args []string) error {
	options := parseOptions(ctx)
	
	log.InitLog("container_commander.log", options.LogPath, true)

	if options.GetGitHash {
		fmt.Println(GitCommitHash)
	} else if options.GetVersion {
		fmt.Println(Version)
	}
	return nil
}
