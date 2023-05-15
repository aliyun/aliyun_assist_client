package checknet

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"time"

	"github.com/aliyun/aliyun_assist_client/agent/util/networkcategory"
)

var (
	ErrNetworkCategoryNotDetected = errors.New("Network category has not been detected")
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
		return 0, errors.New("Failed to find netcheck executable")
	}

	args := []string{"--vpc", "--fast-fail"}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(180)*time.Second)
	defer cancel()

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
