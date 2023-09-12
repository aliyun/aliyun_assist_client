//go:build darwin || freebsd || linux
// +build darwin freebsd linux

package main

import (
	"github.com/aliyun/aliyun_assist_client/thirdparty/aliyun-cli/cli"

	pm "github.com/aliyun/aliyun_assist_client/agent/pluginmanager/acspluginmanager"
	"github.com/aliyun/aliyun_assist_client/agent/pluginmanager/acspluginmanager/flag"
)

func parsePlatformSpecificFlags(fs *cli.FlagSet, executeParams *pm.ExecuteParams) error {
	foreground := flag.ExperimentalForegroundFlag(fs).IsAssigned()

	executeParams.Foreground = foreground
	return nil
}
