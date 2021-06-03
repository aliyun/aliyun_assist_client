package taskengine

import (
  "github.com/stretchr/testify/assert"
  "testing"
)

var strjson string = `{"code":200,"instanceId":"i-2zxxxx","stop":[],"run":[{"task":{"type":"RunBatScript","taskID":"t-bj0sgni1w9lnnk","commandId":"c-bj0sgni1w23ym8","commandContent":"ZWNobyAiMTIzIg==","workingDirectory":null,"args":null,"cron":null,"timeOut":"60","commandSignature":"K06G5B6rDLbvpWesdYpSYDvZrC8tX9nFUjT6izEhOGaxjgDk3ijjWMkpau4D8i1IVN8oxZ4QevdLtzD8MSChqcVWWfUVEWuQzwRhHj/7400DyzwrYz5lk9UuzW3K+dami9AOvx0taONnBAPh0p5Ww6BxGwyRYOAmcDay1pnVoGw=","taskSignature":""},"output":{"interval":3000,"logQuota":24576,"skipEmpty":true,"sendStart":false}}]}`
var strsession string = `{"code":200,"instanceId":"i-2zxxxx","session":[{"cmdContent":"pwd"}]}`


func TestFetchTask(t *testing.T) {
  info := parseTaskInfo(strjson)
  runInfo := info.runInfos[0]

  assert.Equal(t, runInfo.TaskId, "t-bj0sgni1w9lnnk")
  assert.Equal(t, runInfo.CommandType, "RunBatScript")
  assert.Equal(t, runInfo.TimeOut, "60")
  assert.Equal(t, runInfo.Output.SkipEmpty, true)
}

func TestFetchSessionTask(t *testing.T) {
  info := parseTaskInfo(strsession)
  sessionInfo := info.sessionInfos[0]

  assert.Equal(t, sessionInfo.CmdContent, "pwd")
}