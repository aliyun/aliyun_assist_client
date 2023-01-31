package parameters

import (
	"fmt"
	"regexp"

	"github.com/aliyun/aliyun_assist_client/agent/taskengine/taskerrors"
	"github.com/aliyun/aliyun_assist_client/agent/util"
	"github.com/aliyun/aliyun_assist_client/agent/util/networkcategory"
)

var (
	//{{ACS::InstanceId}}
	_environmentParameterPattern = regexp.MustCompile(`{{\s*((?U:ACS)\s*::\s*([\w-.]+))\s*}}`)
)

func ResolveBuiltinParameters(commandContent string, builtinParameters map[string]string) (string, error) {
	// Special treatment when value for builtin parameter "InstanceName" is
	// empty, i.e., some error encountered when luban reads instance name via
	// inner api, and agent needs to try once again but via metaserver
	if valueFromLuban, ok := builtinParameters["InstanceName"]; ok && valueFromLuban == "" {
		if instanceName, err := retrieveInstanceName(); err != nil {
			return "", taskerrors.NewResolvingInstanceNameError(err)
		} else {
			builtinParameters["InstanceName"] = instanceName
		}
	}

	var thrown error = nil
	resolvedContent := _environmentParameterPattern.ReplaceAllStringFunc(commandContent, func(matched string) string {
		match := _environmentParameterPattern.FindStringSubmatch(matched)
		if len(match) != 3 {
			if thrown == nil {
				thrown = taskerrors.NewInvalidEnvironmentParameterError(fmt.Sprintf(`Invalid match %q when resolving environment parameter "%s"`, match, matched))
			}
			return ""
		}

		parameterName := match[2]
		parameterValue, ok := builtinParameters[parameterName]
		if !ok && thrown == nil {
			thrown = taskerrors.NewInvalidEnvironmentParameterError(fmt.Sprintf(`The environment parameter %s is invalid`, parameterName))
		}
		return parameterValue
	})
	if thrown != nil {
		return "", thrown
	}

	return resolvedContent, nil
}

func retrieveInstanceName() (string, error) {
	networkCategory := networkcategory.Get()
	if networkCategory != networkcategory.NetworkVPC &&
		networkCategory != networkcategory.NetworkWithMetaserver {
		return "", fmt.Errorf("Agent is not able to retrieve instance name")
	}

	err, instanceName := util.HttpGet("http://100.100.100.200/latest/meta-data/instance/instance-name")
	return instanceName, err
}
