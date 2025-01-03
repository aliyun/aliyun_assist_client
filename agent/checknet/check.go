package checknet

import (
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

	if err := cmd.Run(); err != nil {
		logger.Error("Netcheck cmd failed: %v", err)
		if exitErr, ok := err.(*exec.ExitError); ok {
			return exitErr.ExitCode(), nil
		}
		return 0, err
	}
	return 0, nil
}
