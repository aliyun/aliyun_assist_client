package main

import (
	"fmt"
	"os"

	"github.com/marcsauter/single"
	"github.com/spf13/pflag"

	"github.com/aliyun/aliyun_assist_client/agent/log"
	"github.com/aliyun/aliyun_assist_client/agent/util"
	"github.com/aliyun/aliyun_assist_client/agent/version"
)

var (
	gitHash   string
	assistVer string
)
var SingleAppLock *single.Single

type Options struct {
	GetHelp bool
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
		fmt.Fprintln(os.Stderr, "Aliyun Assist Copyright (c) 2017-2020 Alibaba Group Holding Limited")
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "Options:")
		pflag.PrintDefaults()
	}

	pflag.Parse()
	return options
}

func main() {
	version.AssistVersion = assistVer
	version.GitCommitHash = gitHash
	// User-Agent header value MUST be manually initialized since version
	// information in version package is manually passed in as above
	util.InitUserAgentValue()

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

	log.InitLog("aliyun_assist_update.log")
	log.GetLogger().Info("Starting ......version:", version.AssistVersion, "githash:", version.GitCommitHash)
	SingleAppLock = single.New("AliyunAssistUpdateSingleLock")
	if err := SingleAppLock.CheckLock(); err != nil && err == single.ErrAlreadyRunning {
		log.GetLogger().Fatal("another instance of the app is already running, exiting")
	}
	defer SingleAppLock.TryUnlock()

	if options.CheckUpdate {
		// Exclusive options check
		if options.ForceUpdate || options.LocalInstall != "" {
			log.GetLogger().Errorln("Invalid multiple options")
			pflag.Usage()
			return
		}

		if err := doCheckUpdate(); err != nil {
			log.GetLogger().Fatalf("Failed to update: %s", err.Error())
		}
	} else if options.ForceUpdate {
		// Exclusive options check
		if options.CheckUpdate || options.LocalInstall != "" {
			log.GetLogger().Errorln("Invalid multiple options")
			pflag.Usage()
			return
		}

		if options.ForceUpdateURL == "" {
			log.GetLogger().Fatalln("ForceUpdate must specify URL via --url option")
			return
		}
		if options.ForceUpdateMD5 == "" {
			log.GetLogger().Fatalln("ForceUpdate must specify md5 via --md5 option")
			return
		}

		log.GetLogger().Infof("Force update: url=%s, md5=%s", options.ForceUpdateURL, options.ForceUpdateMD5)
		if err := doUpdate(options.ForceUpdateURL, options.ForceUpdateMD5, ""); err != nil {
			log.GetLogger().Fatalf("Failure encountered during updating: %s", err.Error())
		}
	} else if options.LocalInstall != "" {
		// Exclusive options check
		if options.CheckUpdate || options.ForceUpdate {
			log.GetLogger().Errorln("Invalid multiple options")
			pflag.Usage()
			return
		}

		if err := doInstall(options.LocalInstall); err != nil {
			log.GetLogger().Fatalf("Failure encountered during executing update script: %s", err.Error())
		}
	} else {
		pflag.Usage()
		return
	}
}
