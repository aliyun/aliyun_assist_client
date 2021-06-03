package statemanager

import (
	"testing"

	"github.com/aliyun/aliyun_assist_client/agent/statemanager/resources"
	"github.com/stretchr/testify/assert"
)

func TestResolve(t *testing.T) {
	paramMap := map[string]interface{}{"regionId": "cn-zhangjiakou", "timeout": 3600}
	v := Resolve(" {{ regionId }} ", paramMap)
	assert.Equal(t, "cn-zhangjiakou", v)

	v = Resolve("ecs.{{ regionId }}.aliyuncs.com/{{regionId}}", paramMap)
	assert.Equal(t, "ecs.cn-zhangjiakou.aliyuncs.com/cn-zhangjiakou", v)

	v = Resolve("{{timeout}}", paramMap)
	assert.Equal(t, 3600, v)

	v = Resolve("{{ timeout }}", paramMap)
	assert.Equal(t, 3600, v)

	v = Resolve("timeout: {{timeout}}", paramMap)
	assert.Equal(t, "timeout: 3600", v)
}

func TestResolveParameterValue(t *testing.T) {
	paramDefs := map[string]ParamProp{
		"regionId": {Type: "String", Default: "cn-hangzhou"},
		"timeout":  {Type: "Number", Default: 60},
		"option":   {Type: "String"},
	}
	userParams := map[string]interface{}{
		"regionId": "cn-zhangjiakou",
		"timeout":  3600,
	}
	params := ResolveParameterValue(paramDefs, userParams)
	assert.Equal(t, "cn-zhangjiakou", params["regionId"])
	assert.Equal(t, 3600, params["timeout"])
	assert.Nil(t, params["option"])
}

func TestResolveJsonParameterValue(t *testing.T) {
	paramDefs := map[string]ParamProp{
		"settings": {
			Type:    "Json",
			Default: map[string]interface{}{
				"key": map[string]interface{}{
					"innerkey" : "innervalue",
				},
			},
		},
	}
	userParams := map[string]interface{}{
		"settings": map[string]interface{}{
			"userkey":"uservalue",
		},
	}

	params := ResolveParameterValue(paramDefs, userParams)
	finalValue := params["settings"].(map[string]interface{})
	assert.Equal(t, "uservalue", finalValue["userkey"])

	params = ResolveParameterValue(paramDefs, map[string]interface{}{})
	finalValue = params["settings"].(map[string]interface{})
	innerValue := finalValue["key"]
	innerJson := innerValue.(map[string]interface{})
	assert.Equal(t, "innervalue", innerJson["innerkey"])
}

var template string = `{
		"FormatVersion": "OOS-2019-06-01-State",
		"Description": "示例模板",
		"Parameters": {
		  "mode": {
			"Type": "String",
			"Default": "644"
		  },
		  "remotePath": {
			"Type": "String"
		  }
		},
		"States": [
		  {
			"ResourceType": "ACS:File",
			"Properties": {
			  "Ensure": "Present",
			  "State": "File",
			  "Mode": "{{ mode }}",
			  "DestinationPath": "/etc/hosts",
			  "SourcePath": "{{remotePath}}/etc/hosts"
			}
		  }
		]
	}`

func TestParseStateTemplate(t *testing.T) {
	templateObj, err := ParseStateTemplate([]byte(template))
	assert.Nil(t, err)
	assert.Equal(t, 2, len(templateObj.Parameters))
	assert.Equal(t, templateObj.Parameters["mode"], ParamProp{
		Type:    "String",
		Default: "644",
	})
	assert.Equal(t, templateObj.Parameters["remotePath"], ParamProp{
		Type: "String",
	})

	assert.Equal(t, 1, len(templateObj.States))
	assert.Equal(t, templateObj.States[0], StateDef{
		ResourceType: "ACS:File",
		Properties: map[string]interface{}{
			"Ensure":          "Present",
			"State":           "File",
			"Mode":            "{{ mode }}",
			"DestinationPath": "/etc/hosts",
			"SourcePath":      "{{remotePath}}/etc/hosts",
		},
	})
}

func TestParseResourceState(t *testing.T) {
	rs, err := ParseResourceState([]byte(template), `{"remotePath":"http://one-oss-bucket"}`)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(rs))
	rs1 := rs[0].(*resources.FileState)
	assert.Equal(t, "Present", rs1.Ensure)
	assert.Equal(t, "File", rs1.State)
	assert.Equal(t, "/etc/hosts", rs1.DestinationPath)
	assert.Equal(t, "644", rs1.Mode)
	assert.Equal(t, "", rs1.Owner)
	assert.Equal(t, "", rs1.Group)
	assert.Equal(t, "http://one-oss-bucket/etc/hosts", rs1.SourcePath)
	assert.Equal(t, "", rs1.Contents)
	assert.Equal(t, "", rs1.Checksum)
	assert.Equal(t, "", rs1.Attributes)
}

var inventoryDataCollectionTemplate string = `{
		"FormatVersion": "OOS-2019-06-01-State",
		"Description": "Inventory data collection",
		"Parameters": {
		  "policy": {
			"Type": "Json"
		  }
		},
		"States": [
		  {
			"ResourceType": "ACS:Inventory",
			"Properties": {
				"Policy": "{{ policy }}"
			}
		  }
		]
	}`

func TestInventoryTemplate(t *testing.T) {
	policy := `
	{
		"policy":
		{
			"ACS:InstanceInformation": {
				"Collection": "Enabled"
			},
			"ACS:File": {
				"Collection": "Enabled",
				"Filters": "[{\"Path\": \"/home/admin/test\",\"Pattern\":[\"*\"],\"Recursive\":false}]"
			}
		}
	}
	`
	rs, err := ParseResourceState([]byte(inventoryDataCollectionTemplate), policy)
	assert.Nil(t, err)

	inventoryPolicy := (rs[0]).(*resources.InventoryState).InventoryPolicy
	assert.Equal(t, "Enabled", inventoryPolicy["ACS:InstanceInformation"].Collection)
	assert.Equal(t, "Enabled", inventoryPolicy["ACS:File"].Collection)
	assert.Equal(t, "[{\"Path\": \"/home/admin/test\",\"Pattern\":[\"*\"],\"Recursive\":false}]", inventoryPolicy["ACS:File"].Filters)
}
