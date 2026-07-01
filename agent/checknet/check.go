package checknet

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/aliyun/aliyun_assist_client/agent/log"
	"github.com/aliyun/aliyun_assist_client/common/networkcategory"
	"github.com/aliyun/aliyun_assist_client/common/requester"
	"github.com/aliyun/aliyun_assist_client/thirdparty/sirupsen/logrus"
	"go.uber.org/atomic"
)

var (
	ErrNetworkCategoryNotDetected = errors.New("Network category has not been detected")
	ErrLastInvokeNotFinish        = errors.New("Last invoke not finish")
	ErrNetcheckExecutableNotFound = errors.New("Failed to find netcheck executable")

	collectionInvokeRunning atomic.Bool
)

func invokeNetcheck() (int, error) {
	networkCategory := networkCategoryCache.Get()
	if networkCategory == networkcategory.NetworkCategoryUnknown {
		return 0, ErrNetworkCategoryNotDetected
	}
	if networkCategory != networkcategory.NetworkVPC {
		return 0, fmt.Errorf("Unsupported network category: %s", string(networkCategory))
	}

	netcheckPath := getNetcheckPath()
	if netcheckPath == "" {
		return 0, ErrNetcheckExecutableNotFound
	}

	args := []string{"stack-diagnostic", "--fast-fail"}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(180)*time.Second)
	defer cancel()

	log.GetLogger().Infof("invokeNetcheck: %s %s", netcheckPath, strings.Join(args, " "))
	cmd := exec.CommandContext(ctx, netcheckPath, args...)
	cmd.Stdin = nil
	cmd.Stdout = nil
	cmd.Stderr = nil

	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return exitErr.ExitCode(), nil
		}

		return 0, err
	}
	return 0, nil
}

type limitedBuffer struct {
	b       bytes.Buffer
	limit   int
}

func NewLimitedBuffer(limit int) *limitedBuffer {
	return &limitedBuffer{
		limit: limit,
	}
}

func (buf *limitedBuffer) Write(p []byte) (int, error) {
	l := len(p)
	if l+buf.b.Len() <= buf.limit {
		buf.b.Write(p)
	} else {
		l = buf.limit - buf.b.Len()
		buf.b.Write(p[:l])
	}
	return len(p), nil
}

func (buf *limitedBuffer) String() string {
	return buf.b.String()
}

// invokeCollection tries to invoke a collection task.
func invokeCollection(logger logrus.FieldLogger, taskId string) (int, error) {
	// There can only be one collection task at a time.
	if !collectionInvokeRunning.CompareAndSwap(false, true) {
		return 0, ErrLastInvokeNotFinish
	}
	defer collectionInvokeRunning.Store(false)

	netcheckPath := getNetcheckPath()
	if netcheckPath == "" {
		return 0, ErrNetcheckExecutableNotFound
	}

	regionId := requester.PeekRegionId()
	regionIdVar := fmt.Sprintf("RegionId=%s", regionId)
	args := []string{"preset-collect", "--confName", "noNetwork", "--variables", regionIdVar}
	if len(taskId) > 0 {
		args = append(args, []string{"--taskId", taskId}...)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(60)*time.Second)
	defer cancel()

	logger.Infof("Invoke netcheck: %s %s", netcheckPath, strings.Join(args, " "))
	cmd := exec.CommandContext(ctx, netcheckPath, args...)

	// When the process exits with a non-zero exit code, the contents of stderr
	// are intercepted as error message, similar to the functionality of
	// exec.ExitError.Stderr.
	// exec.ExitError.Stderr works only when cmd.Output() be called, cmd.Output()
	// buffers all stdout but we not need it, so use a limited stderr buffer
	// instead of exec.ExitError.Stderr.
	stderrBuf := NewLimitedBuffer(1000)
	cmd.Stderr = stderrBuf
	if err := cmd.Run(); err != nil {
		logger.Error("Netcheck cmd failed: %v", err)
		if exitErr, ok := err.(*exec.ExitError); ok {
			return exitErr.ExitCode(), fmt.Errorf(stderrBuf.String())
		}
		return 0, err
	}
	return 0, nil
}
