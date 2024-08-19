package main

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"github.com/aliyun/aliyun_assist_client/agent/log"
	"github.com/aliyun/aliyun_assist_client/interprocess/messagebus/buses"
	messagebus_server "github.com/aliyun/aliyun_assist_client/interprocess/messagebus/server"
	"github.com/aliyun/aliyun_assist_client/plugin/commander/container/idlecheck"
	"github.com/aliyun/aliyun_assist_client/plugin/commander/container/ipc/client"
	"github.com/aliyun/aliyun_assist_client/plugin/commander/container/ipc/endpoint"
	"github.com/aliyun/aliyun_assist_client/plugin/commander/container/ipc/interceptor"
	"github.com/aliyun/aliyun_assist_client/plugin/commander/container/ipc/server"
	"github.com/aliyun/aliyun_assist_client/plugin/commander/container/taskmanager"
	"github.com/aliyun/aliyun_assist_client/thirdparty/aliyun-cli/cli"
	"github.com/aliyun/aliyun_assist_client/thirdparty/aliyun-cli/i18n"
	"github.com/aliyun/aliyun_assist_client/thirdparty/sirupsen/logrus"
	"google.golang.org/grpc"
)

const (
	RunServerSubCmd = "run-server"

	waitTimeFlagName = "waittime"
	pidFileFlagName  = "pidfile"
	endpointFlagName = "endpoint"

	defaultWaitTime = 10
	defaultPidFile  = "container_commander.pid"
)

var (
	runServerFlags = []cli.Flag{
		{
			Name:      pidFileFlagName,
			Shorthand: 'p',
			Short: i18n.T(
				`pid file`,
				`pid文件`,
			),
			AssignedMode: cli.AssignedOnce,
			Category:     "caller",
		},
		{
			Name:      endpointFlagName,
			Shorthand: 'e',
			Short: i18n.T(
				`endpoint`,
				`endpoint`,
			),
			AssignedMode: cli.AssignedOnce,
			Category:     "caller",
		},
		{
			Name:      waitTimeFlagName,
			Shorthand: 'w',
			Short: i18n.T(
				`The time the server is allowed to be idle, default 10s, minimum value is 5s`,
				`允许服务空闲的时间, 默认10秒, 最小5秒`,
			),
			AssignedMode: cli.AssignedOnce,
			Category:     "caller",
		},
	}

	runServerSubCmd = cli.Command{
		Name: RunServerSubCmd,
		Short: i18n.T(
			"Run server process",
			"启动服务进程",
		),
		Usage:             fmt.Sprint(RunServerSubCmd, " [flags]"),
		EnableUnknownFlag: false,
		Run:               runServerCmd,
	}
)

func init() {
	for j := range runServerFlags {
		runServerSubCmd.Flags().Add(&runServerFlags[j])
	}
}

func runServerCmd(ctx *cli.Context, args []string) error {
	var err error
	var waitTime int64
	waitTime = defaultWaitTime
	waitTimeStr, _ := ctx.Flags().Get(waitTimeFlagName).GetValue()
	endpointStr, _ := ctx.Flags().Get(endpointFlagName).GetValue()
	pidFile, _ := ctx.Flags().Get(pidFileFlagName).GetValue()
	logPath, _ := ctx.Flags().Get(LogPathFlagName).GetValue()

	if waitTimeStr != "" {
		waitTime, err = strconv.ParseInt(waitTimeStr, 10, 0)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Invalid param `%s`: %s\n", waitTimeFlagName, waitTimeStr)
			return err
		}
		idlecheck.SetExtendSecond(int(waitTime))
	}

	log.InitLog("container_commander.log", logPath, true)

	go checkIdle(log.GetLogger())

	// write pid file
	if pidFile == "" {
		pidFile = filepath.Join(os.TempDir(), defaultPidFile)
	}
	if err := os.WriteFile(pidFile, []byte(fmt.Sprint(os.Getpid())), 0600); err != nil {
		fmt.Fprintln(os.Stderr, "Write pid file failed: ", err)
		cli.Exit(1)
	}

	var ep *buses.Endpoint
	if endpointStr != "" {
		ep = &buses.Endpoint{}
		if err := ep.Parse(endpointStr); err != nil {
			fmt.Fprintf(os.Stderr, "Invalid endpoint `%s`: %v\n", endpointStr, err)
			cli.Exit(1)
		}
		endpoint.SetEndpoint(ep)
	}

	// register commander to agent
	go func() {
		// wait for server listening
		time.Sleep(time.Second * time.Duration(1))
		endpointStr := os.Getenv("AXT_AGENT_MESSAGEBUS")
		handshakeToken := os.Getenv("AXT_HANDSHAKE_TOKEN")
		endpoint := buses.Endpoint{}
		if err := endpoint.Parse(endpointStr); err != nil {
			log.GetLogger().Errorf("Invalid endpoint from env AXT_AGENT_MESSAGEBUS `%s`: %v", endpointStr, err)
			fmt.Fprintln(os.Stderr, "Invalid endpoint from env AXT_AGENT_MESSAGEBUS: ", endpointStr)
			return
		}
		client.UpdateEndpoint(endpoint)
		if err := client.RegisterCommander(log.GetLogger(), handshakeToken); err != nil {
			log.GetLogger().Error("Register commander failed: ", err)
			fmt.Fprint(os.Stderr, "Register commander failed: ", err)
			return
		}
		fmt.Println("Register commander done")

		log.GetLogger().Info("Register commander done")
	}()
	log.GetLogger().Infof("Starting...... version: %s githash: %s", Version, GitCommitHash)

	signalCh := listenSignal()
	for {
		serveErr := make(chan error)
		server, err := messagebus_server.ListenAndServe(log.GetLogger(), *endpoint.GetEndpoint(true), serveErr,
			[]messagebus_server.RegisterFunc{
				server.RegisterCommanderServer,
			},
			grpc.UnaryInterceptor(
				interceptor.KeepProcessAlive,
			),
		)
		if err != nil {
			log.GetLogger().Error("Listen faield: ", err)
			fmt.Println("Listen failed: ", err)
			cli.Exit(1)
		}
		log.GetLogger().Infof("Listenning on endpoint: %+v", endpoint.GetEndpoint(false).String())
		select {
		case err := <-serveErr:
			log.GetLogger().Error("Server failed: ", err)
			fmt.Println("Server failed: ", err)
			cli.Exit(1)
		case s := <-signalCh:
			if s == syscall.SIGUSR1 {
				// restart grpc server
				log.GetLogger().Info("Recv signal SIGUSR1, restart server")
				server.Stop()
			} else if s == syscall.SIGPIPE {
				log.GetLogger().Warn("Recv signal SIGPIPE, do nothing")
			} else {
				log.GetLogger().Errorf("Recv unhandled signal[%d]", s)
			}
		}
	}
}

func listenSignal() chan os.Signal {
	sigChan := make(chan os.Signal, 2)
	signal.Notify(sigChan, syscall.SIGUSR1, syscall.SIGPIPE)
	return sigChan
}

func checkIdle(logger logrus.FieldLogger) {
	idlecheck.ExtendLive()
	idlecheck.SetChecker("TaskCount", func() bool {
		return taskmanager.TaskCount() == 0
	})
	for {
		when, canExit := idlecheck.TimeToExit()
		if canExit {
			logger.Infof("Server exit actively")
			cli.Exit(0)
		}
		logger.Info("Next check exit when ", when.Format("2006-01-02 15:04:05"))
		<-time.After(time.Until(when))
	}
}
