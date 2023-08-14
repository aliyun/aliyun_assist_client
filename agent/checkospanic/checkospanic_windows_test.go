package checkospanic

import (
	"fmt"
	"io"
	"reflect"
	"testing"
	"time"

	"bou.ke/monkey"
	"github.com/aliyun/aliyun_assist_client/agent/metrics"
	"github.com/aliyun/aliyun_assist_client/agent/util/process"
	"github.com/aliyun/aliyun_assist_client/thirdparty/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestReportLastOsPanic(t *testing.T) {
	var p *process.ProcessCmd
	guard_1 := monkey.PatchInstanceMethod(reflect.TypeOf(p), "SyncRun", func(
		p *process.ProcessCmd,
		workingDir string,
		commandName string,
		commandArguments []string,
		stdoutWriter io.Writer,
		stderrWriter io.Writer,
		stdinReader io.Reader,
		callbackFunc process.ReadCallbackFunc,
		timeOut int) (exitCode int, status int, err error) {
		timeStr := time.Now().Format("01/02/2006 15:04:05")
		content := fmt.Sprintf("createTime:%s \nmessage:计算机已经从检测错误后重新启动。检测错误: 0x000000d1 (0xffff840001612010, 0x0000000000000002, 0x0000000000000000, 0xfffff801710a1981)。已将转储的数据保存在: C:\\Windows\\MEMORY.DMP。...", timeStr)
		stdoutWriter.Write([]byte(content))
		return 0, process.Success, nil
	})
	defer guard_1.Unpatch()
	logger := logrus.NewEntry(logrus.New())
	bugcheck, crashInfo, _ := FindWerSystemErrorReportingEvent(logger)
	fmt.Println("bugcheck: ", bugcheck)
	fmt.Println("crashInfo: ", crashInfo)
	assert.NotEqual(t, bugcheck, "not found")
	assert.NotEqual(t, len(crashInfo), 0)
	var m *metrics.MetricsEvent
	guard_4 := monkey.PatchInstanceMethod(reflect.TypeOf(m), "ReportEvent", func(*metrics.MetricsEvent) {})
	defer guard_4.Unpatch()
	guard_5 := monkey.Patch(metrics.GetWindowsGuestOSPanicEvent, func(keywords ...string) *metrics.MetricsEvent {
		event := &metrics.MetricsEvent{}
		for i := 0; i+1 < len(keywords); i += 2 {
			if len(keywords[i+1]) == 0 {
				fmt.Println(keywords[i])
			}
			assert.NotEqual(t, len(keywords[i+1]), 0)
		}
		return event
	})
	defer guard_5.Unpatch()
	fmt.Println("2")
	ReportLastOsPanic()
}
