//go:build !linux

package main

import (
	"github.com/aliyun/aliyun_assist_client/thirdparty/aliyun-cli/cli"
)

var listContainersCmd *cli.Command