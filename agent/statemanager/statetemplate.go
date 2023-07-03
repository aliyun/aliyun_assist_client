package statemanager

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/aliyun/aliyun_assist_client/agent/log"
	"github.com/aliyun/aliyun_assist_client/agent/statemanager/resources"
	"github.com/aliyun/aliyun_assist_client/thirdparty/sirupsen/logrus"
)

type ParamDef struct {
	Name    string
	Default interface{}
	Value   interface{}
}

type ParamProp struct {
	Type    string
	Default interface{}
}

type StateDef struct {
	ResourceType string
	Properties   map[string]interface{}
}

type StateTemplate struct {
	FormatVersion string
	Parameters    map[string]ParamProp
	States        []StateDef
}

var re *regexp.Regexp

func init() {
	re, _ = regexp.Compile("{{.+?}}")
}

// isRef checks if orig is a reference of parameter
// for example "{{ parameter1 }}"
func isRef(orig string, placeholder string) bool {
	if strings.TrimSpace(orig) == placeholder {
		return true
	}
	return false
}

// Resolve resolves final value for string contains parameter reference
// for example, "ecs.{{ regionId }}.aliyuncs.com" with parameter regionId="cn-zhangjiakou" will get ecs.cn-zhangjiakou.aliyuncs.com
//  "{{id}}" with parameter id=1 will get number 1
func Resolve(orig string, paramMap map[string]interface{}) (v interface{}) {
	placeholders := re.FindAllString(orig, -1)
	if placeholders == nil {
		return orig
	}
	if len(placeholders) == 1 && isRef(orig, placeholders[0]) {
		ph := placeholders[0]
		name := strings.TrimSpace(ph[2 : len(ph)-2])
		paramValue, ok := paramMap[name]
		if !ok {
			return orig
		} else {
			return paramValue
		}
	}
	ret := orig
	for _, ph := range placeholders {
		name := strings.TrimSpace(ph[2 : len(ph)-2])
		paramValue, ok := paramMap[name]
		if ok {
			ret = strings.ReplaceAll(ret, ph, fmt.Sprintf("%v", paramValue))
		}
	}
	return ret
}

func ResolveParameterValue(params map[string]ParamProp, userParams map[string]interface{}) (result map[string]interface{}) {
	if len(params) == 0 {
		return
	}
	result = make(map[string]interface{})
	for name, prop := range params {
		value, ok := userParams[name]
		if ok {
			result[name] = value
		} else if prop.Default != nil {
			result[name] = prop.Default
		} else {
			log.GetLogger().Error("value not specified for parameter" + name)
			result[name] = nil
			continue
		}
	}
	return
}

// ParseStateTemplate parses template data to state and parameter definitions
func ParseStateTemplate(data []byte) (t StateTemplate, err error) {
	err = json.Unmarshal(data, &t)
	if err != nil {
		err = fmt.Errorf("parse template fail: %w", err)
		return
	}
	return
}

func ParseResourceState(data []byte, userParameters string) (rs []resources.ResourceState, err error) {
	template, err := ParseStateTemplate(data)
	if err != nil {
		return
	}
	var parameterValueMap = make(map[string]interface{})
	if strings.TrimSpace(userParameters) != "" {
		err = json.Unmarshal([]byte(userParameters), &parameterValueMap)
		if err != nil {
			log.GetLogger().WithFields(logrus.Fields{
				"parameters": userParameters,
			}).WithError(err).Errorf("parameters is not a valid json")
			return
		}
	}
	paramMap := ResolveParameterValue(template.Parameters, parameterValueMap)
	for _, state := range template.States {
		for k, v := range state.Properties {
			orig, ok := v.(string)
			if ok {
				rv := Resolve(orig, paramMap)
				state.Properties[k] = rv
			}
		}
		resourceState, err2 := NewResourceState(state)
		if err != nil {
			return rs, err2
		}
		rs = append(rs, resourceState)
	}
	return
}
