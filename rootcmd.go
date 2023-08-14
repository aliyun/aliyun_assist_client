package main

import (
	"fmt"
	"os"
	"runtime"
	"runtime/debug"
	"strings"
	"time"

	logrusr "github.com/aliyun/aliyun_assist_client/thirdparty/bombsimon/logrusr/v3"
	"k8s.io/klog/v2"

	"github.com/aliyun/aliyun_assist_client/agent/channel"
	"github.com/aliyun/aliyun_assist_client/agent/checkkdump"
	"github.com/aliyun/aliyun_assist_client/agent/checkospanic"
	"github.com/aliyun/aliyun_assist_client/agent/checkvirt"
	"github.com/aliyun/aliyun_assist_client/agent/clientreport"
	"github.com/aliyun/aliyun_assist_client/agent/cryptdata"
	"github.com/aliyun/aliyun_assist_client/agent/flagging"
	"github.com/aliyun/aliyun_assist_client/agent/heartbeat"
	"github.com/aliyun/aliyun_assist_client/agent/hybrid"
	"github.com/aliyun/aliyun_assist_client/agent/install"
	"github.com/aliyun/aliyun_assist_client/agent/ipc/server"
	"github.com/aliyun/aliyun_assist_client/agent/log"
	"github.com/aliyun/aliyun_assist_client/agent/metrics"
	"github.com/aliyun/aliyun_assist_client/agent/perfmon"
	"github.com/aliyun/aliyun_assist_client/agent/pluginmanager"
	"github.com/aliyun/aliyun_assist_client/agent/statemanager"
	"github.com/aliyun/aliyun_assist_client/agent/taskengine"
	"github.com/aliyun/aliyun_assist_client/agent/taskengine/timermanager"
	"github.com/aliyun/aliyun_assist_client/agent/update"
	"github.com/aliyun/aliyun_assist_client/agent/util"
	"github.com/aliyun/aliyun_assist_client/agent/util/daemon"
	"github.com/aliyun/aliyun_assist_client/agent/util/osutil"
	"github.com/aliyun/aliyun_assist_client/agent/util/wrapgo"
	"github.com/aliyun/aliyun_assist_client/agent/version"
	"github.com/aliyun/aliyun_assist_client/thirdparty/aliyun-cli/cli"
	"github.com/aliyun/aliyun_assist_client/thirdparty/aliyun-cli/i18n"
	"github.com/aliyun/aliyun_assist_client/thirdparty/service"
	"github.com/aliyun/aliyun_assist_client/thirdparty/single"
)

type Options struct {
	GetHelp        bool
	GetVersion     bool
	GetGitHash     bool
	Install        bool
	Remove         bool
	Start          bool
	Stop           bool
	Register       bool
	DeRegister     bool
	Region         string
	Tags           []string
	ActivationCode string
	ActivationId   string
	NetWorkMode    string
	InstanceName   string
	RunAsCommon    bool
	RunAsDaemon    bool
	LogPath        string
	IsVerbose      bool
}

type program struct{}

const (
	HelpFlagName    = "help"
	VersionFlagName = "version"
	GithashFlagName = "githash"
	InstallFlagName = "install"
	RemoveFlagName  = "remove"
	StartFlagName   = "start"
	StopFlagName    = "stop"
	VerboseFlagName = "verbose"

	RegisterFlagName       = "register"
	DeRegisterFlagName     = "deregister"
	RegionFlagName         = "RegionId"
	TagFlagName            = "tag"
	ActivationCodeFlagName = "ActivationCode"
	ActivationIdFlagName   = "ActivationId"
	NetworkModeFlagName    = "NetworkMode"
	InstanceNameFlagName   = "InstanceName"

	LogPathFlagName = "LogPath"

	RunAsCommonFlagName = "common"
	RunAsDaemonFlagName = "daemon"
)

var (
	G_Running     bool          = true
	G_StopEvent   chan struct{} = nil
	SingleAppLock *single.Single

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
			Short:        i18n.T(`log path`, `指定日志保存目录`),
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
		{
			Name:         InstallFlagName,
			Short:        i18n.T(`install assist`, `安装云助手agent为系统服务`),
			AssignedMode: cli.AssignedNone,
			Category:     "caller",
		},
		{
			Name:         RemoveFlagName,
			Short:        i18n.T(`remove assist`, `删除已安装的云助手agent系统服务`),
			AssignedMode: cli.AssignedNone,
			Category:     "caller",
		},
		{
			Name:         StartFlagName,
			Short:        i18n.T(`start assist`, `启动云助手agent系统服务`),
			AssignedMode: cli.AssignedNone,
			Category:     "caller",
		},
		{
			Name:         StopFlagName,
			Short:        i18n.T(`stop assist`, `停止云助手agent系统服务`),
			AssignedMode: cli.AssignedNone,
			Category:     "caller",
		},
		{
			Name:         VerboseFlagName,
			Shorthand:    'V',
			Short:        i18n.T(`enable verbose`, `启用云助手agent的详细运行过程输出`),
			AssignedMode: cli.AssignedNone,
			Category:     "caller",
		},

		{
			Name:         RegisterFlagName,
			Shorthand:    'r',
			Short:        i18n.T(`register as aliyun managed instance`, `注册为云助手托管实例`),
			AssignedMode: cli.AssignedNone,
			Category:     "caller",
		},
		{
			Name:         DeRegisterFlagName,
			Shorthand:    'u',
			Short:        i18n.T(`unregister as aliyun managed instance`, `取消注册为云助手托管实例`),
			AssignedMode: cli.AssignedNone,
			Category:     "caller",
		},
		{
			Name:         RegionFlagName,
			Shorthand:    'R',
			Short:        i18n.T(`used in register mode`, `（该参数仅限在注册为云助手托管实例时使用）`),
			AssignedMode: cli.AssignedOnce,
			Category:     "caller",
		},
		{
			Name:         TagFlagName,
			Shorthand:    'T',
			Short:        i18n.T(`used in register mode`, `（该参数仅限在注册为云助手托管实例时使用）`),
			AssignedMode: cli.AssignedRepeatable,
			Category:     "caller",
		},
		{
			Name:         ActivationCodeFlagName,
			Shorthand:    'C',
			Short:        i18n.T(`used in register mode`, `（该参数仅限在注册为云助手托管实例时使用）`),
			AssignedMode: cli.AssignedOnce,
			Category:     "caller",
		},
		{
			Name:         ActivationIdFlagName,
			Shorthand:    'I',
			Short:        i18n.T(`used in register mode`, `（该参数仅限在注册为云助手托管实例时使用）`),
			AssignedMode: cli.AssignedOnce,
			Category:     "caller",
		},
		{
			Name:         NetworkModeFlagName,
			Shorthand:    'm',
			Short:        i18n.T(`used in register mode`, `（该参数仅限在注册为云助手托管实例时使用）`),
			AssignedMode: cli.AssignedOnce,
			Category:     "caller",
		},
		{
			Name:         InstanceNameFlagName,
			Shorthand:    'N',
			Short:        i18n.T(`used in register mode`, `（该参数仅限在注册为云助手托管实例时使用）`),
			AssignedMode: cli.AssignedOnce,
			Category:     "caller",
		},

		{
			Name:         RunAsCommonFlagName,
			Shorthand:    'c',
			Short:        i18n.T(`run as common`, `以 common 模式运行`),
			AssignedMode: cli.AssignedNone,
			Category:     "caller",
		},
		{
			Name:         RunAsDaemonFlagName,
			Shorthand:    'd',
			Short:        i18n.T(`start as daemon`, `切换到后台运行`),
			AssignedMode: cli.AssignedNone,
			Category:     "caller",
		},
	}

	rootCmd = cli.Command{
		Name:              "aliyun-service",
		Short:             i18n.T(`Aliyun Assist Copyright (c) 2017-2023 Alibaba Group Holding Limited`, `Aliyun Assist Copyright (c) 2017-2023 Alibaba Group Holding Limited`),
		Usage:             "aliyun-service [subcommand] [flags]",
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

func (p *program) Start(s service.Service) error {
	go p.run()
	return nil
}

func (p *program) run() {
	log.GetLogger().Infof("Starting...... version: %s githash: %s", version.AssistVersion, version.GitCommitHash)
	SingleAppLock = single.New("AliyunAssistClientSingleLock")
	if err := SingleAppLock.CheckLock(); err != nil && err == single.ErrAlreadyRunning {
		log.GetLogger().Fatal("another instance of the app is already running, exiting")
	}
	G_Running = true
	G_StopEvent = make(chan struct{})
	channel.TryStartGshellChannel()

	if runtime.GOOS == "windows" {
		util.SetCurrentEnvPath()
	}
	// Logging current working directory information
	if currentWorkingDirectory, err := os.Getwd(); err == nil {
		log.GetLogger().Infof("Current working directory is: %s", currentWorkingDirectory)
	} else {
		log.GetLogger().WithError(err).Errorln("Failed to obtain current working directory")
	}

	sleep_internals_seconds := 3
	for {
		host := util.GetServerHost()
		if host != "" {
			log.GetLogger().Println("GET_HOST_OK ", host)
			break
		} else {
			log.GetLogger().Println("GET_HOST_ERROR")
		}
		time.Sleep(time.Duration(sleep_internals_seconds) * time.Second)
		sleep_internals_seconds = sleep_internals_seconds * 2
		if sleep_internals_seconds > 180 {
			sleep_internals_seconds = 180
		}
	}

	// Use clientreport.LogAndReportPanic as default panic handler to report panic
	wrapgo.SetDefaultPanicHandler(clientreport.LogAndReportPanic)

	// Try to handle panic from code below
	defer func() {
		if panicPayload := recover(); panicPayload != nil {
			stacktrace := debug.Stack()
			wrapgo.CallDefaultPanicHandler(panicPayload, stacktrace)
		}
	}()

	// Check in main goroutine and update as soon as possible, which use stricter
	// timeout limitation. NOTE: The preparation phase timeout parameter should
	// be considered as the whole timeout toleration minus minimum sleeping time
	// for safe updating (5s) minus normal execution time of updating script
	// (usually less than 5s), e.g., 50s - 5s - 5s = 40s.
	if err := update.SafeBootstrapUpdate(time.Duration(40)*time.Second, time.Duration(30)*time.Second); err != nil {
		log.GetLogger().Errorln("Failed to check update when starting: " + err.Error())
		// Failed to update at starting phase would not terminate agent
		// return
	}

	if err := timermanager.InitTimerManager(); err != nil {
		log.GetLogger().Fatalln("Failed to initialize timer manager: " + err.Error())
		return
	}

	if err := update.InitCheckUpdateTimer(); err != nil {
		log.GetLogger().Fatalln("Failed to initialize update checker: " + err.Error())
		metrics.GetUpdateFailedEvent(
			"errormsg", fmt.Sprintf("InitCheckUpdateTimer error: %s", err.Error()),
		).ReportEvent()
		return
	}

	channel.StartChannelMgr()

	if err := heartbeat.InitHeartbeatTimer(); err != nil {
		log.GetLogger().Fatalln("Failed to initialize heartbeat: " + err.Error())
		return
	}

	// TODO: First heart-beat may fail and be failed to indicate agent is ready.
	// Retrying should be tried here.
	heartbeat.PingwithRetries(3)

	if err := statemanager.InitStateManagerTimer(); err != nil {
		log.GetLogger().Errorln("Failed to initialize statemanager: " + err.Error())
	}

	pluginmanager.InitPluginCheckTimer()
	cryptdata.Init()

	if err := checkkdump.CheckKdumpTimer(); err != nil {
		log.GetLogger().Errorln("Failed to StartKdumpCheckTimer: ", err)
	} else {
		log.GetLogger().Infoln("Start StartKdumpCheckTimer")
	}
	server.StartService()

	// Finally, fetching tasks could be allowed and agent starts to run normally.
	taskengine.EnableFetchingTask()
	log.GetLogger().Infoln("Started successfully")
	// And also log to stdout, which would be written to systemd-journal as well
	// as console via systemd
	fmt.Println("Started successfully")
	err := checkvirt.StartVirtIoVersionReport()
	if err != nil {
		log.GetLogger().Errorln("Failed to StartVirtIoVersionReport: " + err.Error())
	} else {
		log.GetLogger().Infoln("Start StartVirtIoVersionReport success")
	}
	// Periodic tasks are retrieved only once at startup
	wrapgo.GoWithDefaultPanicHandler(func() {
		isColdstart, err := flagging.IsColdstart()
		if err != nil {
			log.GetLogger().WithError(err).Errorln("Error encountered when detecting cold-start flag")
		} else {
			startType := "not cold start"
			if isColdstart {
				startType = "cold start"
			}
			metrics.GetBaseStartupEvent(
				"type", startType,
				"osName", osutil.GetVersion(),
			).ReportEvent()
		}

		taskengine.Fetch(false, "", taskengine.NormalTaskType, isColdstart)
	})
	// Report last os panic if panic record found
	if isColdstart, err := flagging.IsColdstart(); err != nil || isColdstart {
		wrapgo.GoWithDefaultPanicHandler(checkospanic.ReportLastOsPanic)
	}

	time.Sleep(time.Duration(3*60) * time.Second)
	log.GetLogger().Infoln("Start PerfMon ......")
	perfmon.StartSelfKillMon()
}

func (p *program) Stop(s service.Service) error {
	log.GetLogger().Println("Stopping ......")
	// channel.StopChannelMgr()
	// //websocket.DisconnectWebsocketServer()
	// G_Running = false
	// close(G_StopEvent)
	// SingleAppLock.TryUnlock()
	// perfmon.StopSelfKillMon()
	log.GetLogger().Println("Stopped")
	return nil
}

func parseOptions(ctx *cli.Context) Options {
	options := Options{}

	options.GetHelp = ctx.Flags().Get(HelpFlagName).IsAssigned()
	options.GetVersion = ctx.Flags().Get(VersionFlagName).IsAssigned()
	options.GetGitHash = ctx.Flags().Get(GithashFlagName).IsAssigned()
	options.Install = ctx.Flags().Get(InstallFlagName).IsAssigned()
	options.Remove = ctx.Flags().Get(RemoveFlagName).IsAssigned()
	options.Start = ctx.Flags().Get(StartFlagName).IsAssigned()
	options.Stop = ctx.Flags().Get(StopFlagName).IsAssigned()
	options.IsVerbose = ctx.Flags().Get(VerboseFlagName).IsAssigned()

	options.Register = ctx.Flags().Get(RegisterFlagName).IsAssigned()
	options.DeRegister = ctx.Flags().Get(DeRegisterFlagName).IsAssigned()
	options.Region, _ = ctx.Flags().Get(RegionFlagName).GetValue()
	options.Tags = ctx.Flags().Get(TagFlagName).GetValues()
	options.ActivationCode, _ = ctx.Flags().Get(ActivationCodeFlagName).GetValue()
	options.ActivationId, _ = ctx.Flags().Get(ActivationIdFlagName).GetValue()
	options.NetWorkMode, _ = ctx.Flags().Get(NetworkModeFlagName).GetValue()
	options.InstanceName, _ = ctx.Flags().Get(InstanceNameFlagName).GetValue()

	options.LogPath, _ = ctx.Flags().Get(LogPathFlagName).GetValue()

	options.RunAsCommon = ctx.Flags().Get(RunAsCommonFlagName).IsAssigned()
	options.RunAsDaemon = ctx.Flags().Get(RunAsDaemonFlagName).IsAssigned()

	return options
}

func runRootCommand(ctx *cli.Context, args []string) error {
	options := parseOptions(ctx)
	log.InitLog("aliyun_assist_main.log", options.LogPath, false)
	// Redirect logging messages from kubernetes CRI client via klog to logrus
	// used by ourselves
	klog.SetLogger(logrusr.New(log.GetLogger()).WithName("klog"))

	if options.LogPath != "" {
		util.SetScriptPath(options.LogPath)
	}
	e := PatchGolang()
	if e != nil {
		log.GetLogger().Fatal("PatchGolang failed :", e.Error())
	}

	if options.IsVerbose {
		util.SetVerboseMode(true)
	}

	if options.GetHelp {
		// aliyun-cli/cli library handles "help" flag internally, and here needs
		// to do nothing.
		return nil
	}
	if options.GetVersion {
		fmt.Println(version.AssistVersion)
		return nil
	}
	if options.GetGitHash {
		fmt.Println(version.GitCommitHash)
		return nil
	}
	if options.Register {
		tags := []hybrid.Tag{}
		for _, tag := range options.Tags {
			words := strings.Split(tag, "=")
			if len(words) != 2 {
				fmt.Println("Invalid tag: ", tag)
				cli.Exit(1)
			}
			tags = append(tags, hybrid.Tag{
				Key:   words[0],
				Value: words[1],
			})
		}
		hybrid.Register(options.Region, options.ActivationCode, options.ActivationId, options.InstanceName, options.NetWorkMode, true, tags)
		return nil
	}
	if options.DeRegister {
		hybrid.UnRegister(true)
		return nil
	}

	if options.RunAsDaemon {
		// TODO: Check other options like --install, --remove, --start, --stop should not be passed
		if err := daemon.Daemonize(); err != nil {
			fmt.Fprintln(os.Stderr, "Failed to start aliyun-service as daemon:", err)
		}
		return nil
	}

	svcConfig := install.ServiceConfig()
	prg := &program{}
	s, err := service.New(prg, svcConfig)
	if err != nil {
		fmt.Println("new service error " + err.Error())
		return nil
	}

	if options.Stop {
		if err := s.Stop(); err != nil {
			fmt.Println("stop assist failed:", err)
		} else {
			fmt.Println("stop assist ok")
		}
		return nil
	}

	if options.Remove {
		if err := s.Uninstall(); err != nil {
			fmt.Println("uninstall assist failed:", err)
		} else {
			fmt.Println("uninstall assist ok")
		}
		return nil
	}

	if options.Install {
		if err := s.Install(); err != nil {
			fmt.Println("install assist failed:", err)
		} else {
			fmt.Println("install assist ok")
		}
		return nil
	}

	if options.Start {
		if err := s.Start(); err != nil {
			fmt.Println("start assist failed:", err)
		} else {
			fmt.Println("start assist ok")
		}
		return nil
	}
	err = s.Run()
	if err != nil {
		log.GetLogger().Println(err.Error())
	}

	return nil
}
