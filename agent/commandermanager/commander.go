package commandermanager

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/aliyun/aliyun_assist_client/agent/commandermanager/processutil"
	"github.com/aliyun/aliyun_assist_client/agent/log"
	"github.com/aliyun/aliyun_assist_client/agent/taskengine/taskerrors"
	pb "github.com/aliyun/aliyun_assist_client/interprocess/commander/agrpc"
	commander_client "github.com/aliyun/aliyun_assist_client/interprocess/commander/client"
	"github.com/aliyun/aliyun_assist_client/interprocess/messagebus/buses"
	messagebus_client "github.com/aliyun/aliyun_assist_client/interprocess/messagebus/client"
	"github.com/aliyun/aliyun_assist_client/thirdparty/sirupsen/logrus"
	"github.com/shirou/gopsutil/v3/process"

	"google.golang.org/grpc"
)

type Commander struct {
	config *CommanderConfig

	l      sync.Mutex
	logger logrus.FieldLogger

	client     *commander_client.CommanderClient
	clientLock sync.Mutex

	exitFunc   map[string]func()
	onExitLock sync.Mutex

	handshakeDone     chan error
	handshakeDoneLock sync.Mutex
	handshakeToken    string
}

type CommanderConfig struct {
	CommanderName string
	CmdPath       string

	PidFile    string
	Endpoint   buses.Endpoint
	Version    string
	ApiVersion string

	// timeout to wait commander process start and finish handshake
	StartTimeout time.Duration
	// timeout to finish handshake with exist commander process
	AttachTimeout time.Duration
	// timeout to connect commander's grpc server
	DialTimeout time.Duration
}

func NewCommander(config *CommanderConfig) *Commander {
	c := &Commander{
		config: config,
		logger: log.GetLogger().WithField("commander", config.CommanderName),
	}
	return c
}

// RegisterExitHandler register callback functions which be called after
// commander process exit
func (c *Commander) RegisterExitHandler(name string, f func()) {
	c.onExitLock.Lock()
	defer c.onExitLock.Unlock()

	if c.exitFunc == nil {
		c.exitFunc = make(map[string]func())
	}
	c.exitFunc[name] = f
}

func (c *Commander) DeRegisterExitHandler(name string) {
	c.onExitLock.Lock()
	defer c.onExitLock.Unlock()

	delete(c.exitFunc, name)
}

// CmdPath return commander's executable path
func (c *Commander) CmdPath() string {
	return c.config.CmdPath
}

// // CmdPath return commander's version
func (c *Commander) Version() string {
	return c.config.Version
}

// Client return commander's grpc client
func (c *Commander) Client() (*commander_client.CommanderClient, *taskerrors.CommanderError) {
	c.clientLock.Lock()
	defer c.clientLock.Unlock()

	if c.client == nil {
		return nil, taskerrors.NewClientNotCreatedError("client is nil")
	}
	return c.client, nil
}

// IsRunning check if commander is running
func (c *Commander) IsRunning() bool {
	c.clientLock.Lock()
	defer c.clientLock.Unlock()

	if c.client == nil {
		return false
	}
	return c.ping() == nil
}

func (c *Commander) ping() error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	resp, err := c.client.Ping(ctx)
	if err != nil {
		c.logger.Error("Ping failed: ", err)
		return err
	} else if resp.Status.StatusCode != 0 {
		c.logger.Error("Ping failed: ", resp.Status.ErrMessage)
		return fmt.Errorf(resp.Status.ErrMessage)
	}
	return nil
}

// CreateClient create a new grpc client
func (c *Commander) CreateClient(endpoint buses.Endpoint) (*commander_client.CommanderClient, *taskerrors.CommanderError) {
	c.clientLock.Lock()
	defer c.clientLock.Unlock()

	conn, err := messagebus_client.ConnectWithTimeout(c.logger, endpoint, c.config.DialTimeout)
	if err != nil {
		return nil, taskerrors.NewCreateClientFailedError(err.Error())
	}
	client := commander_client.NewClient(conn)

	if c.client != nil {
		c.client.Close()
	}
	c.client = client

	return c.client, nil
}

// Start start commander
func (c *Commander) Start(token string) *taskerrors.CommanderError {
	c.l.Lock()
	defer c.l.Unlock()

	if p, alive := c.checkProcessAlive(); alive {
		if err := c.attachProcess(int(p.Pid), token); err != nil {
			c.logger.Error("Attach to existed process failed: ", err)
			if err := p.Kill(); err != nil {
				c.logger.Error("Kill existed process failed: ", err)
			}
			if err := c.startProcess(token); err != nil {
				c.logger.Error("Start process failed: ", err)
				return err
			} else {
				c.logger.Info("Process started!")
			}
		} else {
			c.logger.Info("Attached to existed process!")
		}
	} else {
		if err := c.startProcess(token); err != nil {
			c.logger.Error("Start process failed: ", err)
			return err
		} else {
			c.logger.Info("Process started!")
		}
	}
	return nil
}

// checkProcessAlive check if commander process is running
func (c *Commander) checkProcessAlive() (*process.Process, bool) {
	content, err := os.ReadFile(c.config.PidFile)
	if err != nil {
		return nil, false
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(content)))
	if err != nil {
		return nil, false
	}
	p, err := process.NewProcess(int32(pid))
	if err != nil {
		return nil, false
	}
	cmdlineSlice, err := p.CmdlineSlice()
	c.logger.Info("commander cmdline: ", cmdlineSlice)
	if err != nil || len(cmdlineSlice) == 0 || strings.TrimSpace(cmdlineSlice[0]) != c.config.CmdPath {
		return nil, false
	}
	return p, true
}

// startProcess start commander process
func (c *Commander) startProcess(token string) (processErr *taskerrors.CommanderError) {
	c.logger.Info("Try to start commander process")

	endpoint := buses.GetCentralEndpoint(false)
	args := []string{"run-server", "--pidfile", c.config.PidFile, "--endpoint", c.config.Endpoint.String()}
	cmd := exec.Command(c.config.CmdPath, args...)
	cmd.Env = append(cmd.Env, os.Environ()...)
	cmd.Env = append(cmd.Env, fmt.Sprintf("AXT_AGENT_MESSAGEBUS=%s", endpoint.String()))

	c.handshakeToken = token
	c.handshakeDoneLock.Lock()
	c.handshakeDone = make(chan error, 1)
	c.handshakeDoneLock.Unlock()
	cmd.Env = append(cmd.Env, fmt.Sprintf("AXT_HANDSHAKE_TOKEN=%s", c.handshakeToken))
	defer func() {
		c.handshakeToken = ""
		close(c.handshakeDone)
		c.handshakeDoneLock.Lock()
		defer c.handshakeDoneLock.Unlock()
		c.handshakeDone = nil
	}()
	c.logger.Info(strings.Join(cmd.Args, " "))

	// Agent take over the stdout and stderr of commander process, so if agent
	// process stops commander process will receive SIGPIPE in linux. So
	// commander process in linux need to handle SIGPIPE to prevent unexpected exit.
	cmdStdout, err := cmd.StdoutPipe()
	if err != nil {
		return taskerrors.NewStartProcessFailedError(fmt.Sprintf("create stdout pipe failed, %v", err))
	}
	cmdStderr, err := cmd.StderrPipe()
	if err != nil {
		return taskerrors.NewStartProcessFailedError(fmt.Sprintf("create stderr pipe failed, %v", err))
	}
	if err = cmd.Start(); err != nil {
		return taskerrors.NewStartProcessFailedError(fmt.Sprintf("start process failed, %v", err))
	}

	c.logger = c.logger.WithField("commander-pid", cmd.Process.Pid)

	// Make sure the command is properly cleaned up if there is an error
	defer func() {
		if r := recover(); r != nil {
			processErr = taskerrors.NewStartProcessFailedError(fmt.Sprintf("panic occurred, %v", err))
			cmdStdout.Close()
			cmdStderr.Close()
		}
	}()

	var stdouterrWaitGroup sync.WaitGroup
	// wait stdout/stderr read goroutine
	stdouterrWaitGroup.Add(1)
	stdouterrWaitGroup.Add(1)

	exitCtx, ctxCancel := context.WithCancel(context.Background())
	go func() {
		// wait to finish reading from stdout/stderr since the stdout/stderr
		// pipe reader will be closed by the subsequent call to cmd.Wait().
		stdouterrWaitGroup.Wait()

		// Wait for the command to end.
		err := cmd.Wait()
		if err != nil {
			c.logger.WithError(err).Error("commander process exited")
		} else {
			c.logger.Info("commander process exited")
		}

		ctxCancel()
		c.onExit()
	}()

	timeout := time.After(c.config.StartTimeout)
	stdoutScanner := bufio.NewScanner(cmdStdout)
	stderrScanner := bufio.NewScanner(cmdStderr)
	c.logger.Info("Waiting for connection")
	go func() {
		defer func() {
			stdouterrWaitGroup.Done()
			cmdStdout.Close()
		}()
		logger := c.logger.WithField("from", "stdout")
		for stdoutScanner.Scan() {
			logger.Info(stdoutScanner.Text())
		}
	}()
	go func() {
		defer func() {
			stdouterrWaitGroup.Done()
			cmdStderr.Close()
		}()
		logger := c.logger.WithField("from", "stderr")
		for stderrScanner.Scan() {
			logger.Info(stderrScanner.Text())
		}
	}()

	select {
	case err = <-c.handshakeDone:
		if err != nil {
			processErr = taskerrors.NewStartProcessFailedError(fmt.Sprintf("handshake failed, %v", err))
		}
	case <-exitCtx.Done():
		processErr = taskerrors.NewStartProcessFailedError("process exited before handshake finishe")
	case <-timeout:
		processErr = taskerrors.NewStartProcessFailedError("timeout while commander start")
		cmd.Process.Kill()
	}

	return processErr
}

// attachProcess attach running commander process
func (c *Commander) attachProcess(pid int, token string) *taskerrors.CommanderError {
	c.logger.Info("Try attach commander process")

	errMsg := []string{}
	conn, err := messagebus_client.ConnectWithTimeout(c.logger, c.config.Endpoint, c.config.DialTimeout)
	if err != nil {
		c.logger.WithError(err).Error("Connect failed, try to restart commander grpc server")
		errMsg = append(errMsg, fmt.Sprintf("connection failed, %v", err))
		if err := processutil.RestartServer(pid); err != nil {
			c.logger.WithError(err).Error("Restart commander grpc server failed")
			errMsg = append(errMsg, fmt.Sprintf("restart server failed, %v", err))
		}
		time.Sleep(time.Second)
		conn, err = messagebus_client.ConnectWithTimeout(c.logger, c.config.Endpoint, c.config.DialTimeout)
		if err != nil {
			c.logger.WithError(err).Error("Retry connect failed")
			errMsg = append(errMsg, fmt.Sprintf("retry connection failed, %v", err))
			return taskerrors.NewAttachProcessFailedError(errMsg...)
		}
	}
	defer conn.Close()

	if err := c.handshake(conn, token); err != nil {
		return err
	}
	c.logger = c.logger.WithField("commander-pid", pid)

	go func() {
		processutil.PidWait(pid)
		c.onExit()
	}()

	return nil
}

func (c *Commander) handshake(conn *grpc.ClientConn, token string) *taskerrors.CommanderError {
	client := commander_client.NewClient(conn)
	defer client.Close()

	c.handshakeToken = token
	c.handshakeDoneLock.Lock()
	c.handshakeDone = make(chan error, 1)
	c.handshakeDoneLock.Unlock()
	defer func() {
		c.handshakeToken = ""
		close(c.handshakeDone)
		c.handshakeDoneLock.Lock()
		defer c.handshakeDoneLock.Unlock()
		c.handshakeDone = nil
	}()

	agentEndpoint := buses.GetCentralEndpoint(false)
	req := &pb.HandshakeReq{
		Endpoint:       agentEndpoint.String(),
		HandshakeToken: c.handshakeToken,
	}
	ctx, cancel := context.WithTimeout(context.Background(), c.config.AttachTimeout)
	defer cancel()
	resp, err := client.Client.Handshake(ctx, req)
	if err != nil {
		return taskerrors.NewAttachProcessFailedError(fmt.Sprintf("handshake request fail, %v", err))
	} else if resp.Status.StatusCode != 0 {
		return taskerrors.NewAttachProcessFailedError(fmt.Sprintf("handshake request fail, %s", resp.Status.ErrMessage))
	}

	select {
	case <-ctx.Done():
		return taskerrors.NewAttachProcessFailedError("attach commander timeout")
	case err := <-c.handshakeDone:
		if err != nil {
			return taskerrors.NewStartProcessFailedError(fmt.Sprintf("handshake failed, %v", err))
		}
	}

	return nil
}

func (c *Commander) onExit() {
	c.logger.Info("commander process exited")
	c.logger = c.logger.WithField("commander-pid", "nil")

	// reset c.conn
	c.clientLock.Lock()
	if c.client != nil {
		c.client.Close()
	}
	c.clientLock.Unlock()

	c.onExitLock.Lock()
	defer c.onExitLock.Unlock()
	for k, f := range c.exitFunc {
		c.logger.Infof("call function %s when commander exit", k)
		f()
	}
}

// HandShakeDone tell startProcess/attachProcess that handshake finished
func (c *Commander) HandShakeDone(token string) {
	c.handshakeDoneLock.Lock()
	defer c.handshakeDoneLock.Unlock()

	if c.handshakeDone != nil {
		// c.handshakeDone may be closed before put err into it, this will make
		// panic
		defer func() {
			if err := recover(); err != nil {
				c.logger.WithField("func", "HandShakeDone").Error(err)
			}
		}()
		var err error
		if token == "" {
			err = fmt.Errorf("invalid handshake token")
		} else if token != c.handshakeToken {
			err = fmt.Errorf("handshake token not match, want[%s] but [%s]", c.handshakeToken, token)
		}
		c.handshakeDone <- err
	}
}
