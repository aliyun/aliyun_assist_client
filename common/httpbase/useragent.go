package httpbase

import (
	"fmt"
	"runtime"

	"github.com/aliyun/aliyun_assist_client/agent/version"
)

const (
	UserAgentHeader = "User-Agent"
)

var (
	UserAgentValue string = fmt.Sprintf("%s_%s/%s", runtime.GOOS, runtime.GOARCH, version.AssistVersion)
)
