package main

import (
	"fmt"
	"os"

	"github.com/marcsauter/single"
	"github.com/spf13/pflag"

	"github.com/aliyun/aliyun_assist_client/agent/log"
	"github.com/aliyun/aliyun_assist_client/agent/version"
	"github.com/aliyun/aliyun_assist_client/common/envutil"
)

var SingleAppLock *single.Single

type Options struct {
	GetHelp    bool
	GetVersion bool
	GetGitHash bool

	// Exclusive group: Check update option
	CheckUpdate bool

	// Exclusive group: Force update options
	ForceUpdate bool
	// Required options for force update
	ForceUpdateURL string
	ForceUpdateMD5 string

	// Exclusive group: Local install (should be invisible for general users)
	LocalInstall string
}

func parseOptions() Options {
	options := Options{}
	pflag.BoolVarP(&options.GetHelp, "help", "h", false, "print help")
	pflag.BoolVarP(&options.GetVersion, "version", "v", false, "print version")
	pflag.BoolVar(&options.GetGitHash, "githash", false, "print git hash")

	// Exclusive group: Check update option
	pflag.BoolVarP(&options.CheckUpdate, "check_update", "c", false, "Check and update if necessary")

	// Exclusive group: Force update options
	pflag.BoolVarP(&options.ForceUpdate, "force_update", "f", false, "Force update with specified package")
	pflag.StringVarP(&options.ForceUpdateURL, "url", "u", "", "Download URL of specified update package")
	pflag.StringVarP(&options.ForceUpdateMD5, "md5", "m", "", "MD5 checksum of specified update package")

	// Exclusive group: Local install
	pflag.StringVar(&options.LocalInstall, "local_install", "", "Invoke local install script of extracted update package")

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
	options := parseOptions()

	if options.GetHelp {
		pflag.Usage()
		return
	}
	if options.GetVersion {
		fmt.Println(version.AssistVersion)
		return
	}
	if options.GetGitHash {
		fmt.Println(version.GitCommitHash)
		return
	}

	log.InitLog("aliyun_assist_update.log", "", false)
	log.GetLogger().Infof("Starting...... version: %s githash: %s", version.AssistVersion, version.GitCommitHash)
	SingleAppLock = single.New("AliyunAssistUpdateSingleLock")
	if err := SingleAppLock.CheckLock(); err != nil && err == single.ErrAlreadyRunning {
		fmt.Fprintln(os.Stderr, "Error: another instance of the app is already running")
		os.Exit(1)
		return
	}
	defer SingleAppLock.TryUnlock()

	envutil.ClearExecErrDot()
	if options.CheckUpdate {
		// Exclusive options check
		if options.ForceUpdate || options.LocalInstall != "" {
			fmt.Fprintln(os.Stderr, "Invalid options: specified options in conflict")
			pflag.Usage()
			os.Exit(1)
			return
		}

		if err := doCheckUpdate(); err != nil {
			log.GetLogger().WithError(err).Errorln("Failed to check or update agent")
			fmt.Fprintln(os.Stderr, "Error:", err.Error())
			os.Exit(1)
		}
		return
	} else if options.ForceUpdate {
		// Exclusive options check
		if options.CheckUpdate || options.LocalInstall != "" {
			fmt.Fprintln(os.Stderr, "Invalid options: specified options in conflict")
			pflag.Usage()
			os.Exit(1)
			return
		}

		if options.ForceUpdateURL == "" {
			fmt.Fprintln(os.Stderr, "Invalid options: -f/--force_update option needs specifying update package URL via --url option")
			pflag.Usage()
			os.Exit(1)
			return
		}
		if options.ForceUpdateMD5 == "" {
			fmt.Fprintln(os.Stderr, "Invalid options: -f/--force_update option needs specifying MD5 checksum of update package via --md5 option")
			pflag.Usage()
			os.Exit(1)
			return
		}

		log.GetLogger().Infof("Force update: url=%s, md5=%s", options.ForceUpdateURL, options.ForceUpdateMD5)
		if err := doUpdate(options.ForceUpdateURL, options.ForceUpdateMD5, ""); err != nil {
			log.GetLogger().WithError(err).Errorln("Failed to perform force updating")
			fmt.Fprintln(os.Stderr, "Error:", err.Error())
			os.Exit(1)
		}
		return
	} else if options.LocalInstall != "" {
		// Exclusive options check
		if options.CheckUpdate || options.ForceUpdate {
			fmt.Fprintln(os.Stderr, "Invalid options: specified options in conflict")
			pflag.Usage()
			os.Exit(1)
			return
		}

		if err := doInstall(options.LocalInstall); err != nil {
			log.GetLogger().WithError(err).Errorln("Failed to execute update script")
			fmt.Fprintln(os.Stderr, "Error:", err.Error())
			os.Exit(1)
			return
		}
	} else {
		pflag.Usage()
		return
	}
}
