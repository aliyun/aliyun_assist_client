package acspluginmanager

import (
	"strings"

	"github.com/google/shlex"
)

type ExecFetchOptions struct {
	File string

	PluginName string
	PluginId   string
	Version    string
	Local      bool

	FetchTimeoutInSeconds int
}

type VerifyFetchOptions struct {
	Url string

	FetchTimeoutInSeconds int
}

type CommonExecuteParams struct {
	Params    string
	Separator string
	ParamsV2  string

	OptionalExecutionTimeoutInSeconds *int
}

func (ep *CommonExecuteParams) SplitArgs() []string {
	var args []string
	if ep.ParamsV2 != "" {
		args, _ = shlex.Split(ep.ParamsV2)
	} else {
		if ep.Separator == "" {
			ep.Separator = ","
		}
		paramsSpace := strings.Replace(ep.Params, ep.Separator, " ", -1)
		args, _ = shlex.Split(paramsSpace)
	}

	if len(args) == 0 {
		args = nil
	}
	return args
}
