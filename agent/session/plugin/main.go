package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/aliyun/aliyun_assist_client/agent/session/plugin/cli"
	"github.com/aliyun/aliyun_assist_client/agent/session/plugin/config"
	"github.com/aliyun/aliyun_assist_client/agent/session/plugin/log"
	"github.com/aliyun/aliyun_assist_client/agent/session/plugin/portforward"
	"github.com/aliyun/aliyun_assist_client/agent/session/plugin/sendpublickey"
	"github.com/aliyun/aliyun_assist_client/agent/session/plugin/session"
	"github.com/aliyun/aliyun_assist_client/agent/session/plugin/ssh"
	"github.com/aliyun/aliyun_assist_client/agent/session/plugin/i18n"

	"github.com/spf13/pflag"
)

var (
	gitHash string
	cliVer  string = "10.0.0.1"
)

type Options struct {
	GetHelp      bool
	GetVersion   bool
	WebsocketUrl string
	IsVerbose    bool
	IsRawMode    bool
}

func parseOptions() Options {
	options := Options{}
	pflag.BoolVarP(&options.GetHelp, "help", "h", false, "print help")
	pflag.BoolVarP(&options.GetVersion, "version", "v", false, "print version")

	pflag.StringVarP(&options.WebsocketUrl, "WebsocketUrl", "u", "", "WebsocketUrl")

	pflag.BoolVarP(&options.IsVerbose, "verbose", "V", false, "enable verbose")
	pflag.BoolVarP(&options.IsRawMode, "rawmode", "r", false, "enable rawmode")

	pflag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n", os.Args[0])
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "Aliyun Assist Copyright (c) 2017-2023 Alibaba Group Holding Limited")
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "Options:")
		pflag.PrintDefaults()
	}

	pflag.Parse()
	return options
}

func main() {
	log.InitLog("aliyun_ecs_session_log", "", true)
	log.GetLogger().Infoln("session plugin started_ ", os.Args[:])

	cli.PlatformCompatible()
	cli.Version = cliVer
	writer := cli.DefaultWriter()
	stderr := cli.DefaultStderrWriter()

	// load current configuration
	profile, err := config.LoadCurrentProfile()
	if err != nil {
		cli.Errorf(stderr, "ERROR: load current configuration failed %s", err)
		return
	}

	// set language with current profile
	i18n.SetLanguage(profile.Language)

	// create root command
	rootCmd := &cli.Command{
		Name:              "ali-instance-cli",
		Short:             i18n.T("Alibaba Cloud ECS DevOps Command Line Interface Version "+cli.Version, "阿里云ecs运维CLI命令行工具 "+cli.Version),
		Usage:             "ali-instance-cli <operation> [--parameter1 value1 --parameter2 value2 ...]",
		Sample:            "ali-instance-cli session <instance_id>",
		EnableUnknownFlag: true,
	}

	// add default flags
	config.AddFlags(rootCmd.Flags())
	//openapi.AddFlags(rootCmd.Flags())

	// new open api commando to process rootCmd
	//commando := openapi.NewCommando(writer, profile)
	//commando.InitWithCommand(rootCmd)

	ctx := cli.NewCommandContext(writer, stderr)
	ctx.EnterCommand(rootCmd)
	ctx.SetCompletion(cli.ParseCompletionForShell())

	rootCmd.AddSubCommand(config.NewConfigureCommand())
	//rootCmd.AddSubCommand(lib.NewOssCommand())
	rootCmd.AddSubCommand(cli.NewVersionCommand())
	rootCmd.AddSubCommand(session.NewSessionCommand())
	rootCmd.AddSubCommand(ssh.NewSshCommand())
	rootCmd.AddSubCommand(portforward.NewPortForwardCommand())
	rootCmd.AddSubCommand(sendpublickey.NewSendPublicKeyCommand())
	//rootCmd.AddSubCommand(cli.NewAutoCompleteCommand())
	rootCmd.Execute(ctx, os.Args[1:])
}

func waitSignals() error {
	sigChan := make(chan os.Signal, 2)
	signal.Notify(
		sigChan,
		syscall.SIGINT,
		syscall.SIGTERM,
	)

	<-sigChan
	log.GetLogger().Infoln("session plugin stop", sigChan)
	os.Exit(1)

	return nil
}
