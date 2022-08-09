package parameters

import (
	"fmt"
	"regexp"
)

type InvalidEnvironmentParameterError struct {
	ParameterName string
}

var (
	//{{ACS::InstanceId}}
	_environmentParameterPattern = regexp.MustCompile(`{{\s*((?U:ACS)\s*::\s*([\w-.]+))\s*}}`)
)

func NewInvalidEnvironmentParameterError(parameterName string) *InvalidEnvironmentParameterError {
	return &InvalidEnvironmentParameterError{
		ParameterName: parameterName,
	}
}

func (pe *InvalidEnvironmentParameterError) Error() string {
	return fmt.Sprintf("The environment parameter %s is invalid", pe.ParameterName)
}

func ResolveEnvironmentParameters(commandContent string, environmentArguments map[string]string) (string, error) {
	var thrown error = nil
	resolvedContent := _environmentParameterPattern.ReplaceAllStringFunc(commandContent, func(matched string) string {
		match := _environmentParameterPattern.FindStringSubmatch(matched)
		if len(match) != 3 {
			if thrown == nil {
				thrown = fmt.Errorf(`Invalid match %q when resolving environment parameter "%s"`, match, matched)
			}
			return ""
		}

		parameterName := match[2]
		parameterValue, ok := environmentArguments[parameterName]
		if !ok && thrown == nil {
			thrown = NewInvalidEnvironmentParameterError(parameterName)
		}
		return parameterValue
	})
	if thrown != nil {
		return "", thrown
	}

	return resolvedContent, nil
}
