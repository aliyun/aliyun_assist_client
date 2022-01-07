package main

import (
	"fmt"
	"os"
	"runtime"
	"runtime/debug"
	"time"

	"github.com/aliyun/aliyun_assist_client/agent/checkvirt"
	"github.com/aliyun/aliyun_assist_client/agent/pluginmanager"

	"github.com/spf13/pflag"

	"github.com/aliyun/aliyun_assist_client/thirdparty/service"
	"github.com/aliyun/aliyun_assist_client/thirdparty/single"

	"github.com/aliyun/aliyun_assist_client/agent/channel"
	"github.com/aliyun/aliyun_assist_client/agent/clientreport"
	"github.com/aliyun/aliyun_assist_client/agent/flagging"
	"github.com/aliyun/aliyun_assist_client/agent/heartbeat"
	"github.com/aliyun/aliyun_assist_client/agent/hybrid"
	"github.com/aliyun/aliyun_assist_client/agent/install"
	"github.com/aliyun/aliyun_assist_client/agent/log"
	"github.com/aliyun/aliyun_assist_client/agent/metrics"
	"github.com/aliyun/aliyun_assist_client/agent/perfmon"
	"github.com/aliyun/aliyun_assist_client/agent/statemanager"
	"github.com/aliyun/aliyun_assist_client/agent/taskengine"
	"github.com/aliyun/aliyun_assist_client/agent/taskengine/timermanager"
	"github.com/aliyun/aliyun_assist_client/agent/update"
	"github.com/aliyun/aliyun_assist_client/agent/util"
	"github.com/aliyun/aliyun_assist_client/agent/util/daemon"
	"github.com/aliyun/aliyun_assist_client/agent/util/osutil"
	"github.com/aliyun/aliyun_assist_client/agent/util/wrapgo"
	"github.com/aliyun/aliyun_assist_client/agent/version"
)

var G_Running bool = true
var G_StopEvent chan struct{} = nil
var G_IsWindows bool = false
var G_IsFreebsd bool = false
var G_IsLinux bool = false
var SingleAppLock *single.Single

var (
	gitHash   string
	assistVer string = "10.10.10.10000"
)

type program struct{}

func init() {
	if runtime.GOOS == "windows" {
		G_IsWindows = true
	} else if runtime.GOOS == "linux" {
		G_IsLinux = true
	} else if runtime.GOOS == "freebsd" {
		G_IsFreebsd = true
	} else {

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

	if succ := pluginmanager.InitPluginCheckTimer(); succ == false {
		log.GetLogger().Errorln("plugin check timer fail")
	}

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
	ActivationCode string
	ActivationId   string
	InstanceName   string
	RunAsCommon    bool
	RunAsDaemon    bool
	LogPath        string
	IsVerbose      bool
}

func parseOptions() Options {
	options := Options{}
	pflag.BoolVarP(&options.GetHelp, "help", "h", false, "print help")
	pflag.BoolVarP(&options.GetVersion, "version", "v", false, "print version")
	pflag.BoolVar(&options.GetGitHash, "githash", false, "print git hash")
	pflag.BoolVar(&options.Install, "install", false, "install assist")
	pflag.BoolVar(&options.Remove, "remove", false, "remove assist")
	pflag.BoolVar(&options.Start, "start", false, "start assist")
	pflag.BoolVar(&options.Stop, "stop", false, "stop assist")
	pflag.BoolVarP(&options.IsVerbose, "verbose", "V", false, "enable verbose")

	pflag.BoolVarP(&options.Register, "register", "r", false, "register as aliyun managed instance")
	pflag.BoolVarP(&options.DeRegister, "deregister", "u", false, "unregister as aliyun managed instance")
	pflag.StringVarP(&options.Region, "RegionId", "R", "", "used in register mode")
	pflag.StringVarP(&options.ActivationCode, "ActivationCode", "C", "", "used in register mode")
	pflag.StringVarP(&options.ActivationId, "ActivationId", "I", "", "used in register mode")
	pflag.StringVarP(&options.InstanceName, "InstanceName", "N", "", "used in register mode")

	pflag.StringVarP(&options.LogPath, "LogPath", "L", "", "log path")

	pflag.BoolVarP(&options.RunAsCommon, "common", "c", false, "run as common")
	pflag.BoolVarP(&options.RunAsDaemon, "daemon", "d", false, "start as daemon")

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
	options := parseOptions()
	log.InitLog("aliyun_assist_main.log", options.LogPath)
	if options.LogPath != "" {
		util.SetScriptPath(options.LogPath)
	}
	e := PatchGolang()
	if e != nil {
		log.GetLogger().Fatal("PatchGolang failed :", e.Error())
	}
	version.AssistVersion = assistVer
	version.GitCommitHash = gitHash
	// User-Agent header value MUST be manually initialized since version
	// information in version package is manually passed in as above
	util.InitUserAgentValue()

	if options.IsVerbose {
		util.SetVerboseMode(true)
	}

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
	if options.Register {
		hybrid.Register(options.Region, options.ActivationCode, options.ActivationId, options.InstanceName, true)
		return
	}
	if options.DeRegister {
		hybrid.UnRegister(true)
		return
	}

	if options.RunAsDaemon {
		// TODO: Check other options like --install, --remove, --start, --stop should not be passed
		if err := daemon.Daemonize(); err != nil {
			fmt.Fprintln(os.Stderr, "Failed to start aliyun-service as daemon:", err)
		}
		return
	}

	svcConfig := install.ServiceConfig()
	prg := &program{}
	s, err := service.New(prg, svcConfig)
	if err != nil {
		fmt.Println("new service error " + err.Error())
		return
	}

	if options.Stop {
		if err := s.Stop(); err != nil {
			fmt.Println("stop assist failed:", err)
		} else {
			fmt.Println("stop assist ok")
		}
		return
	}

	if options.Remove {
		if err := s.Uninstall(); err != nil {
			fmt.Println("uninstall assist failed:", err)
		} else {
			fmt.Println("uninstall assist ok")
		}
		return
	}

	if options.Install {
		if err := s.Install(); err != nil {
			fmt.Println("install assist failed:", err)
		} else {
			fmt.Println("install assist ok")
		}
		return
	}

	if options.Start {
		if err := s.Start(); err != nil {
			fmt.Println("start assist failed:", err)
		} else {
			fmt.Println("start assist ok")
		}
		return
	}
	err = s.Run()
	if err != nil {
		log.GetLogger().Println(err.Error())
	}
}
