package util

import (
	"errors"
	"regexp"
	"strings"
)

//{{oos-secret:db_password}}
var oosParamPattern = regexp.MustCompile("{{\\s*((?U:oos-secret|oos)\\s*:\\s*[\\w-.]+)\\s*}}")

func isValidParameterStore(param string) bool {
	return oosParamPattern.MatchString(param)
}

func ReplaceAllParameterStore(param string) (string, error) {
	result := param
	for {
		matchGroups := oosParamPattern.FindStringSubmatch(result)
		if matchGroups == nil {
			return result, nil
		}
		value, err := replaceParameterStoreValue(result, matchGroups)
		if err != nil {
			return value, err
		}
		result = value
	}
}

func getParameterStoreValue(matchGroups []string) (string, error) {
	if matchGroups == nil || len(matchGroups) != 2 {
		return "", errors.New("Invalid matchGroups")
	}
	if !isValidParameterStore(matchGroups[0]) {
		return "", errors.New("Invalid matchGroups[0]")
	}
	parts := strings.Split(matchGroups[1], ":")
	if len(parts) != 2 {
		return "", errors.New("Invalid matchGroups[1]")
	}
	paraName := strings.TrimSpace(parts[1])
	ParamType := strings.TrimSpace(parts[0])
	if ParamType == "oos-secret" {
		return GetSecretParam(paraName)
	} else if ParamType == "oos" {
		return GetParam(paraName)
	} else {
		return "", errors.New("Invalid ParamType")
	}
}

func replaceParameterStoreValue(param string, matchGroups []string) (string, error) {
	value, err := getParameterStoreValue(matchGroups)
	if err != nil {
		return value, err
	}
	result := strings.Replace(param, matchGroups[0], value, 1)
	return result, nil
}
