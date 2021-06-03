package main

import (
	"fmt"
	"github.com/aliyun/aliyun_assist_client/agent/log"
	client "github.com/aliyun/aliyun_assist_client/agent/session/plugin"
	"github.com/aliyun/aliyun_assist_client/agent/util"
	"github.com/spf13/pflag"
	"os"

	"github.com/sirupsen/logrus"
)

const pluginVersion  = "1.0"

type Options struct {
	GetHelp        bool
	GetVersion     bool
	WebsocketUrl            string
	IsVerbose     bool
}

func parseOptions() Options {
	options := Options{}
	pflag.BoolVarP(&options.GetHelp, "help", "h", false, "print help")
	pflag.BoolVarP(&options.GetVersion, "version", "v", false, "print version")

	pflag.StringVarP(&options.WebsocketUrl, "WebsocketUrl", "u", "", "WebsocketUrl")

	pflag.BoolVarP(&options.IsVerbose, "verbose", "V", false, "enable verbose")

	pflag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n", os.Args[0])
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "Aliyun Assist Copyright (c) 2017-2020 Alibaba Group Holding Limited")
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "Options:")
		pflag.PrintDefaults()
	}

	pflag.Parse()
	return options
}
func main() {
	log.InitLog("aliyun_ecs_session_log", "")
    log.GetLogger().Infoln("session plugin started")
	options := parseOptions()

	if options.IsVerbose {
		util.SetVerboseMode(true)
	}
	if options.GetHelp {
		pflag.Usage()
		return
	}
	if options.GetVersion {
		fmt.Println(pluginVersion)
		return
	}

	if options.WebsocketUrl != "" {
		url := options.WebsocketUrl
		client, err := client.NewClient(url, "")
		// loop
		if err = client.Loop(); err != nil {
			logrus.Fatalf("Communication error: %v", err)
		}
		return
	}

}


