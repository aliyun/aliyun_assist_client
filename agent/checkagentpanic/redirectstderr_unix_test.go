package checkagentpanic

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	gomonkey "github.com/agiledragon/gomonkey/v2"
	"github.com/aliyun/aliyun_assist_client/agent/metrics"
	"github.com/aliyun/aliyun_assist_client/agent/util"
	"github.com/stretchr/testify/assert"
)

const (
	mokePanicTime = "2023-08-02 20:53:28"
)

func TestCheckAgentPanic(t *testing.T) {
	guard_IsSystemdLinux := gomonkey.ApplyFunc(util.IsSystemdLinux, func() bool { return true })
	guard_ExecCmd := gomonkey.ApplyFunc(util.ExeCmd, func(cmd string) (error, string, string) {
		fmt.Println(cmd)
		if strings.Index(cmd, "since") == -1 {
			return nil, `{"SYSLOG_FACILITY":"3","PRIORITY":"6","_MACHINE_ID":"263ecb5f3a73425da7e3b92798e9b155","_PID":"13044","SYSLOG_IDENTIFIER":"aliyun-service","_COMM":"aliyun-service","_TRANSPORT":"stdout","_CMDLINE":"/usr/local/share/aliyun-assist/2.4.3.421/aliyun-service","MESSAGE":"Agent check point","__CURSOR":"s=12a775a969ac4a54b7ad6b7a6f7fcc94;i=5adc9;b=95bc8a874d734fe08ebe334910033bbc;m=44274171;t=601f02612a06d;x=fc33f471bc67b44b","_BOOT_ID":"95bc8a874d734fe08ebe334910033bbc","_SYSTEMD_UNIT":"aliyun.service","_SYSTEMD_SLICE":"system.slice","__MONOTONIC_TIMESTAMP":"1143423345","_GID":"0","_EXE":"/usr/local/share/aliyun-assist/2.4.3.421/aliyun-service","_SYSTEMD_CGROUP":"/system.slice/aliyun.service","_STREAM_ID":"7eda98c07a024491846ecc92eacfe757","_SELINUX_CONTEXT":"unconfined\n","_UID":"0","_HOSTNAME":"ubuntu22","_CAP_EFFECTIVE":"1ffffffffff","_SYSTEMD_INVOCATION_ID":"05de3634af0642b7aa8f4ffc4dab59d9","__REALTIME_TIMESTAMP":"1690980802797677"}`, ""
		} else {
			return nil, `{"__CURSOR":"s=12a775a969ac4a54b7ad6b7a6f7fcc94;i=5adc9;b=95bc8a874d734fe08ebe334910033bbc;m=44274171;t=601f02612a06d;x=fc33f471bc67b44b","_SYSTEMD_CGROUP":"/system.slice/aliyun.service","_EXE":"/usr/local/share/aliyun-assist/2.4.3.421/aliyun-service","_SYSTEMD_SLICE":"system.slice","_HOSTNAME":"ubuntu22","_GID":"0","_CAP_EFFECTIVE":"1ffffffffff","MESSAGE":"Agent check point","SYSLOG_FACILITY":"3","_TRANSPORT":"stdout","_STREAM_ID":"7eda98c07a024491846ecc92eacfe757","_PID":"13044","_SYSTEMD_INVOCATION_ID":"05de3634af0642b7aa8f4ffc4dab59d9","PRIORITY":"6","_COMM":"aliyun-service","_CMDLINE":"/usr/local/share/aliyun-assist/2.4.3.421/aliyun-service","_MACHINE_ID":"263ecb5f3a73425da7e3b92798e9b155","_UID":"0","_SELINUX_CONTEXT":"unconfined\n","_BOOT_ID":"95bc8a874d734fe08ebe334910033bbc","SYSLOG_IDENTIFIER":"aliyun-service","__REALTIME_TIMESTAMP":"1690980802797677","_SYSTEMD_UNIT":"aliyun.service","__MONOTONIC_TIMESTAMP":"1143423345"}
{"__CURSOR":"s=12a775a969ac4a54b7ad6b7a6f7fcc94;i=5adca;b=95bc8a874d734fe08ebe334910033bbc;m=447a0daf;t=601f026656ca0;x=163880f50e711224","__REALTIME_TIMESTAMP":"1690980808223904","_PID":"13044","_SYSTEMD_INVOCATION_ID":"05de3634af0642b7aa8f4ffc4dab59d9","_HOSTNAME":"ubuntu22","SYSLOG_FACILITY":"3","MESSAGE":"panic: runtime error: invalid memory address or nil pointer dereference","_SELINUX_CONTEXT":"unconfined\n","_EXE":"/usr/local/share/aliyun-assist/2.4.3.421/aliyun-service","_SYSTEMD_UNIT":"aliyun.service","_STREAM_ID":"7eda98c07a024491846ecc92eacfe757","_SYSTEMD_SLICE":"system.slice","PRIORITY":"6","__MONOTONIC_TIMESTAMP":"1148849583","_COMM":"aliyun-service","_MACHINE_ID":"263ecb5f3a73425da7e3b92798e9b155","SYSLOG_IDENTIFIER":"aliyun-service","_SYSTEMD_CGROUP":"/system.slice/aliyun.service","_GID":"0","_CMDLINE":"/usr/local/share/aliyun-assist/2.4.3.421/aliyun-service","_CAP_EFFECTIVE":"1ffffffffff","_TRANSPORT":"stdout","_UID":"0","_BOOT_ID":"95bc8a874d734fe08ebe334910033bbc"}
{"_CMDLINE":"/usr/local/share/aliyun-assist/2.4.3.421/aliyun-service","_SYSTEMD_INVOCATION_ID":"05de3634af0642b7aa8f4ffc4dab59d9","_HOSTNAME":"ubuntu22","_SYSTEMD_SLICE":"system.slice","SYSLOG_IDENTIFIER":"aliyun-service","_TRANSPORT":"stdout","_MACHINE_ID":"263ecb5f3a73425da7e3b92798e9b155","_EXE":"/usr/local/share/aliyun-assist/2.4.3.421/aliyun-service","_SYSTEMD_CGROUP":"/system.slice/aliyun.service","_PID":"13044","_UID":"0","MESSAGE":"goroutine 34 [running]:","_COMM":"aliyun-service","PRIORITY":"6","__CURSOR":"s=12a775a969ac4a54b7ad6b7a6f7fcc94;i=5adcb;b=95bc8a874d734fe08ebe334910033bbc;m=447a0daf;t=601f026656ca0;x=a2b0446cd8bcb93a","__MONOTONIC_TIMESTAMP":"1148849583","__REALTIME_TIMESTAMP":"1690980808223904","SYSLOG_FACILITY":"3","_STREAM_ID":"7eda98c07a024491846ecc92eacfe757","_SELINUX_CONTEXT":"unconfined\n","_CAP_EFFECTIVE":"1ffffffffff","_BOOT_ID":"95bc8a874d734fe08ebe334910033bbc","_SYSTEMD_UNIT":"aliyun.service","_GID":"0"}
{"SYSLOG_FACILITY":"3","_SYSTEMD_SLICE":"system.slice","_PID":"13044","_SYSTEMD_INVOCATION_ID":"05de3634af0642b7aa8f4ffc4dab59d9","SYSLOG_IDENTIFIER":"aliyun-service","_UID":"0","__MONOTONIC_TIMESTAMP":"1148849583","_CAP_EFFECTIVE":"1ffffffffff","__REALTIME_TIMESTAMP":"1690980808223904","_CMDLINE":"/usr/local/share/aliyun-assist/2.4.3.421/aliyun-service","_SELINUX_CONTEXT":"unconfined\n","_BOOT_ID":"95bc8a874d734fe08ebe334910033bbc","_SYSTEMD_CGROUP":"/system.slice/aliyun.service","PRIORITY":"6","_EXE":"/usr/local/share/aliyun-assist/2.4.3.421/aliyun-service","_SYSTEMD_UNIT":"aliyun.service","_STREAM_ID":"7eda98c07a024491846ecc92eacfe757","_HOSTNAME":"ubuntu22","__CURSOR":"s=12a775a969ac4a54b7ad6b7a6f7fcc94;i=5adcc;b=95bc8a874d734fe08ebe334910033bbc;m=447a0daf;t=601f026656ca0;x=c8e4f0ff2fce8d2d","_GID":"0","_MACHINE_ID":"263ecb5f3a73425da7e3b92798e9b155","_TRANSPORT":"stdout","_COMM":"aliyun-service","MESSAGE":"runtime/debug.Stack()"}
{"SYSLOG_FACILITY":"3","MESSAGE":"\t/root/.g/versions/1.20.5/src/runtime/debug/stack.go:24 +0x64","_SYSTEMD_SLICE":"system.slice","_SELINUX_CONTEXT":"unconfined\n","_HOSTNAME":"ubuntu22","PRIORITY":"6","_TRANSPORT":"stdout","_CAP_EFFECTIVE":"1ffffffffff","_COMM":"aliyun-service","_STREAM_ID":"7eda98c07a024491846ecc92eacfe757","_GID":"0","_UID":"0","_EXE":"/usr/local/share/aliyun-assist/2.4.3.421/aliyun-service","_CMDLINE":"/usr/local/share/aliyun-assist/2.4.3.421/aliyun-service","_SYSTEMD_CGROUP":"/system.slice/aliyun.service","_MACHINE_ID":"263ecb5f3a73425da7e3b92798e9b155","_PID":"13044","_SYSTEMD_UNIT":"aliyun.service","_SYSTEMD_INVOCATION_ID":"05de3634af0642b7aa8f4ffc4dab59d9","__MONOTONIC_TIMESTAMP":"1148849583","__REALTIME_TIMESTAMP":"1690980808223904","_BOOT_ID":"95bc8a874d734fe08ebe334910033bbc","SYSLOG_IDENTIFIER":"aliyun-service","__CURSOR":"s=12a775a969ac4a54b7ad6b7a6f7fcc94;i=5adcd;b=95bc8a874d734fe08ebe334910033bbc;m=447a0daf;t=601f026656ca0;x=b99db48f0c5fb7d6"}
{"_SYSTEMD_UNIT":"aliyun.service","SYSLOG_IDENTIFIER":"aliyun-service","_COMM":"aliyun-service","SYSLOG_FACILITY":"3","_SYSTEMD_INVOCATION_ID":"05de3634af0642b7aa8f4ffc4dab59d9","_CMDLINE":"/usr/local/share/aliyun-assist/2.4.3.421/aliyun-service","PRIORITY":"6","_HOSTNAME":"ubuntu22","_SELINUX_CONTEXT":"unconfined\n","_GID":"0","_SYSTEMD_SLICE":"system.slice","_UID":"0","_EXE":"/usr/local/share/aliyun-assist/2.4.3.421/aliyun-service","_PID":"13044","_BOOT_ID":"95bc8a874d734fe08ebe334910033bbc","_SYSTEMD_CGROUP":"/system.slice/aliyun.service","MESSAGE":"main.(*program).run.func1()","__REALTIME_TIMESTAMP":"1690980808223904","_TRANSPORT":"stdout","__MONOTONIC_TIMESTAMP":"1148849583","_CAP_EFFECTIVE":"1ffffffffff","_MACHINE_ID":"263ecb5f3a73425da7e3b92798e9b155","_STREAM_ID":"7eda98c07a024491846ecc92eacfe757","__CURSOR":"s=12a775a969ac4a54b7ad6b7a6f7fcc94;i=5adce;b=95bc8a874d734fe08ebe334910033bbc;m=447a0daf;t=601f026656ca0;x=94c4c924cb59094"}
{"_CAP_EFFECTIVE":"1ffffffffff","SYSLOG_FACILITY":"3","_SYSTEMD_SLICE":"system.slice","MESSAGE":"\t/root/go/aliyun_assist_client/rootcmd.go:301 +0x34","_PID":"13044","_EXE":"/usr/local/share/aliyun-assist/2.4.3.421/aliyun-service","_SYSTEMD_CGROUP":"/system.slice/aliyun.service","__REALTIME_TIMESTAMP":"1690980808223904","_CMDLINE":"/usr/local/share/aliyun-assist/2.4.3.421/aliyun-service","_COMM":"aliyun-service","_SYSTEMD_INVOCATION_ID":"05de3634af0642b7aa8f4ffc4dab59d9","_BOOT_ID":"95bc8a874d734fe08ebe334910033bbc","PRIORITY":"6","_MACHINE_ID":"263ecb5f3a73425da7e3b92798e9b155","_UID":"0","SYSLOG_IDENTIFIER":"aliyun-service","_SYSTEMD_UNIT":"aliyun.service","__MONOTONIC_TIMESTAMP":"1148849583","_STREAM_ID":"7eda98c07a024491846ecc92eacfe757","_HOSTNAME":"ubuntu22","__CURSOR":"s=12a775a969ac4a54b7ad6b7a6f7fcc94;i=5adcf;b=95bc8a874d734fe08ebe334910033bbc;m=447a0daf;t=601f026656ca0;x=de3f022ca63f800d","_TRANSPORT":"stdout","_GID":"0","_SELINUX_CONTEXT":"unconfined\n"}
{"SYSLOG_FACILITY":"3","_STREAM_ID":"7eda98c07a024491846ecc92eacfe757","_COMM":"aliyun-service","__REALTIME_TIMESTAMP":"1690980808223904","_SELINUX_CONTEXT":"unconfined\n","_EXE":"/usr/local/share/aliyun-assist/2.4.3.421/aliyun-service","_GID":"0","_SYSTEMD_SLICE":"system.slice","_PID":"13044","_SYSTEMD_INVOCATION_ID":"05de3634af0642b7aa8f4ffc4dab59d9","_MACHINE_ID":"263ecb5f3a73425da7e3b92798e9b155","_TRANSPORT":"stdout","SYSLOG_IDENTIFIER":"aliyun-service","PRIORITY":"6","__CURSOR":"s=12a775a969ac4a54b7ad6b7a6f7fcc94;i=5add0;b=95bc8a874d734fe08ebe334910033bbc;m=447a0daf;t=601f026656ca0;x=9389ee63b31232ca","_CAP_EFFECTIVE":"1ffffffffff","_UID":"0","_SYSTEMD_CGROUP":"/system.slice/aliyun.service","_SYSTEMD_UNIT":"aliyun.service","_HOSTNAME":"ubuntu22","_BOOT_ID":"95bc8a874d734fe08ebe334910033bbc","MESSAGE":"panic({0xa3e120, 0x14f9ae0})","__MONOTONIC_TIMESTAMP":"1148849583","_CMDLINE":"/usr/local/share/aliyun-assist/2.4.3.421/aliyun-service"}
{"_STREAM_ID":"7eda98c07a024491846ecc92eacfe757","_SYSTEMD_INVOCATION_ID":"05de3634af0642b7aa8f4ffc4dab59d9","_SYSTEMD_UNIT":"aliyun.service","_TRANSPORT":"stdout","_PID":"13044","_GID":"0","SYSLOG_FACILITY":"3","_SYSTEMD_SLICE":"system.slice","_SELINUX_CONTEXT":"unconfined\n","_BOOT_ID":"95bc8a874d734fe08ebe334910033bbc","_MACHINE_ID":"263ecb5f3a73425da7e3b92798e9b155","_HOSTNAME":"ubuntu22","_SYSTEMD_CGROUP":"/system.slice/aliyun.service","_EXE":"/usr/local/share/aliyun-assist/2.4.3.421/aliyun-service","_UID":"0","_COMM":"aliyun-service","__REALTIME_TIMESTAMP":"1690980808223904","__MONOTONIC_TIMESTAMP":"1148849583","MESSAGE":"\t/root/.g/versions/1.20.5/src/runtime/panic.go:884 +0x1f4","_CMDLINE":"/usr/local/share/aliyun-assist/2.4.3.421/aliyun-service","PRIORITY":"6","_CAP_EFFECTIVE":"1ffffffffff","SYSLOG_IDENTIFIER":"aliyun-service","__CURSOR":"s=12a775a969ac4a54b7ad6b7a6f7fcc94;i=5add1;b=95bc8a874d734fe08ebe334910033bbc;m=447a0daf;t=601f026656ca0;x=46775d46beadb584"}
{"_UID":"0","SYSLOG_IDENTIFIER":"aliyun-service","_MACHINE_ID":"263ecb5f3a73425da7e3b92798e9b155","_BOOT_ID":"95bc8a874d734fe08ebe334910033bbc","_SYSTEMD_SLICE":"system.slice","MESSAGE":"github.com/aliyun/aliyun_assist_client/agent/pluginmanager.InitPluginCheckTimer()","_TRANSPORT":"stdout","_EXE":"/usr/local/share/aliyun-assist/2.4.3.421/aliyun-service","__CURSOR":"s=12a775a969ac4a54b7ad6b7a6f7fcc94;i=5add2;b=95bc8a874d734fe08ebe334910033bbc;m=447a0daf;t=601f026656ca0;x=f0f07ce5df9d2516","__MONOTONIC_TIMESTAMP":"1148849583","_SYSTEMD_INVOCATION_ID":"05de3634af0642b7aa8f4ffc4dab59d9","_COMM":"aliyun-service","_GID":"0","_SYSTEMD_UNIT":"aliyun.service","__REALTIME_TIMESTAMP":"1690980808223904","_PID":"13044","_CAP_EFFECTIVE":"1ffffffffff","_SYSTEMD_CGROUP":"/system.slice/aliyun.service","_CMDLINE":"/usr/local/share/aliyun-assist/2.4.3.421/aliyun-service","SYSLOG_FACILITY":"3","_HOSTNAME":"ubuntu22","_STREAM_ID":"7eda98c07a024491846ecc92eacfe757","PRIORITY":"6","_SELINUX_CONTEXT":"unconfined\n"}
{"_SYSTEMD_INVOCATION_ID":"05de3634af0642b7aa8f4ffc4dab59d9","_HOSTNAME":"ubuntu22","SYSLOG_FACILITY":"3","_MACHINE_ID":"263ecb5f3a73425da7e3b92798e9b155","_STREAM_ID":"7eda98c07a024491846ecc92eacfe757","_CMDLINE":"/usr/local/share/aliyun-assist/2.4.3.421/aliyun-service","__MONOTONIC_TIMESTAMP":"1148849583","_TRANSPORT":"stdout","_SYSTEMD_CGROUP":"/system.slice/aliyun.service","_BOOT_ID":"95bc8a874d734fe08ebe334910033bbc","PRIORITY":"6","_SYSTEMD_UNIT":"aliyun.service","MESSAGE":"\t/root/go/aliyun_assist_client/agent/pluginmanager/pluginmanager.go:64 +0x10c","_CAP_EFFECTIVE":"1ffffffffff","_SELINUX_CONTEXT":"unconfined\n","__CURSOR":"s=12a775a969ac4a54b7ad6b7a6f7fcc94;i=5add3;b=95bc8a874d734fe08ebe334910033bbc;m=447a0daf;t=601f026656ca0;x=c6a211eb61a74fa","__REALTIME_TIMESTAMP":"1690980808223904","_UID":"0","_SYSTEMD_SLICE":"system.slice","SYSLOG_IDENTIFIER":"aliyun-service","_GID":"0","_PID":"13044","_EXE":"/usr/local/share/aliyun-assist/2.4.3.421/aliyun-service","_COMM":"aliyun-service"}
{"__MONOTONIC_TIMESTAMP":"1148849583","_HOSTNAME":"ubuntu22","_MACHINE_ID":"263ecb5f3a73425da7e3b92798e9b155","_UID":"0","_COMM":"aliyun-service","__REALTIME_TIMESTAMP":"1690980808223904","SYSLOG_FACILITY":"3","_SYSTEMD_SLICE":"system.slice","_SYSTEMD_CGROUP":"/system.slice/aliyun.service","_GID":"0","_SELINUX_CONTEXT":"unconfined\n","_CAP_EFFECTIVE":"1ffffffffff","__CURSOR":"s=12a775a969ac4a54b7ad6b7a6f7fcc94;i=5add4;b=95bc8a874d734fe08ebe334910033bbc;m=447a0daf;t=601f026656ca0;x=d8b568749285f949","_SYSTEMD_INVOCATION_ID":"05de3634af0642b7aa8f4ffc4dab59d9","PRIORITY":"6","_EXE":"/usr/local/share/aliyun-assist/2.4.3.421/aliyun-service","_CMDLINE":"/usr/local/share/aliyun-assist/2.4.3.421/aliyun-service","_BOOT_ID":"95bc8a874d734fe08ebe334910033bbc","_PID":"13044","_SYSTEMD_UNIT":"aliyun.service","_STREAM_ID":"7eda98c07a024491846ecc92eacfe757","MESSAGE":"main.(*program).run(0x0?)","SYSLOG_IDENTIFIER":"aliyun-service","_TRANSPORT":"stdout"}
{"SYSLOG_FACILITY":"3","PRIORITY":"6","__REALTIME_TIMESTAMP":"1690980808223904","_SYSTEMD_UNIT":"aliyun.service","_STREAM_ID":"7eda98c07a024491846ecc92eacfe757","_SYSTEMD_SLICE":"system.slice","_CMDLINE":"/usr/local/share/aliyun-assist/2.4.3.421/aliyun-service","__CURSOR":"s=12a775a969ac4a54b7ad6b7a6f7fcc94;i=5add5;b=95bc8a874d734fe08ebe334910033bbc;m=447a0daf;t=601f026656ca0;x=4a15c9443bd443f8","_HOSTNAME":"ubuntu22","SYSLOG_IDENTIFIER":"aliyun-service","_SYSTEMD_CGROUP":"/system.slice/aliyun.service","_EXE":"/usr/local/share/aliyun-assist/2.4.3.421/aliyun-service","_SELINUX_CONTEXT":"unconfined\n","_UID":"0","_PID":"13044","_TRANSPORT":"stdout","_CAP_EFFECTIVE":"1ffffffffff","__MONOTONIC_TIMESTAMP":"1148849583","_MACHINE_ID":"263ecb5f3a73425da7e3b92798e9b155","MESSAGE":"\t/root/go/aliyun_assist_client/rootcmd.go:345 +0x8a4","_COMM":"aliyun-service","_BOOT_ID":"95bc8a874d734fe08ebe334910033bbc","_GID":"0","_SYSTEMD_INVOCATION_ID":"05de3634af0642b7aa8f4ffc4dab59d9"}
{"SYSLOG_IDENTIFIER":"aliyun-service","_COMM":"aliyun-service","_CMDLINE":"/usr/local/share/aliyun-assist/2.4.3.421/aliyun-service","_BOOT_ID":"95bc8a874d734fe08ebe334910033bbc","_SYSTEMD_SLICE":"system.slice","_SELINUX_CONTEXT":"unconfined\n","_SYSTEMD_UNIT":"aliyun.service","__CURSOR":"s=12a775a969ac4a54b7ad6b7a6f7fcc94;i=5add6;b=95bc8a874d734fe08ebe334910033bbc;m=447a0daf;t=601f026656ca0;x=76742e76725d70b1","PRIORITY":"6","_GID":"0","_SYSTEMD_INVOCATION_ID":"05de3634af0642b7aa8f4ffc4dab59d9","_STREAM_ID":"7eda98c07a024491846ecc92eacfe757","_MACHINE_ID":"263ecb5f3a73425da7e3b92798e9b155","_SYSTEMD_CGROUP":"/system.slice/aliyun.service","_CAP_EFFECTIVE":"1ffffffffff","MESSAGE":"created by main.(*program).Start","_TRANSPORT":"stdout","_HOSTNAME":"ubuntu22","_UID":"0","_EXE":"/usr/local/share/aliyun-assist/2.4.3.421/aliyun-service","_PID":"13044","SYSLOG_FACILITY":"3","__MONOTONIC_TIMESTAMP":"1148849583","__REALTIME_TIMESTAMP":"1690980808223904"}
{"__REALTIME_TIMESTAMP":"1690980808223904","PRIORITY":"6","_TRANSPORT":"stdout","SYSLOG_FACILITY":"3","_GID":"0","_EXE":"/usr/local/share/aliyun-assist/2.4.3.421/aliyun-service","_CAP_EFFECTIVE":"1ffffffffff","_UID":"0","__CURSOR":"s=12a775a969ac4a54b7ad6b7a6f7fcc94;i=5add7;b=95bc8a874d734fe08ebe334910033bbc;m=447a0daf;t=601f026656ca0;x=98a083a8161a2d8b","_HOSTNAME":"ubuntu22","_CMDLINE":"/usr/local/share/aliyun-assist/2.4.3.421/aliyun-service","_SYSTEMD_SLICE":"system.slice","_BOOT_ID":"95bc8a874d734fe08ebe334910033bbc","__MONOTONIC_TIMESTAMP":"1148849583","MESSAGE":"\t/root/go/aliyun_assist_client/rootcmd.go:254 +0x5c","_STREAM_ID":"7eda98c07a024491846ecc92eacfe757","_COMM":"aliyun-service","_SYSTEMD_INVOCATION_ID":"05de3634af0642b7aa8f4ffc4dab59d9","SYSLOG_IDENTIFIER":"aliyun-service","_MACHINE_ID":"263ecb5f3a73425da7e3b92798e9b155","_SYSTEMD_UNIT":"aliyun.service","_SELINUX_CONTEXT":"unconfined\n","_PID":"13044","_SYSTEMD_CGROUP":"/system.slice/aliyun.service"}`, ""
		}
	})
	defer guard_ExecCmd.Reset()
	var reported bool
	var m *metrics.MetricsEvent
	guard_metricsReportEvent := gomonkey.ApplyMethod(reflect.TypeOf(m), "ReportEvent", func(event *metrics.MetricsEvent) {
		reported = true
		keywords := make(map[string]string)
		err := json.Unmarshal([]byte(event.KeyWords), &keywords)
		assert.Equal(t, nil, err)
		panicTime, ok := keywords["panicTime"]
		assert.Equal(t, true, ok)
		assert.Equal(t, mokePanicTime, panicTime)
		panicInfo, ok := keywords["panicInfo"]
		assert.Equal(t, true, ok)
		assert.NotEqual(t, "", panicInfo)
		// fmt.Println(panicTime)
		// fmt.Println(panicInfo)
	})
	defer guard_metricsReportEvent.Reset()
	err := CheckAgentPanic()
	assert.Equal(t, nil, err)
	assert.Equal(t, true, reported)
	guard_IsSystemdLinux.Reset()
	// fo not systemd
	mokeLogFile := "./moke.log"
	logcontent := `Agent check point
panic: runtime error: invalid memory address or nil pointer dereference
goroutine 34 [running]:
runtime/debug.Stack()
        /root/.g/versions/1.20.5/src/runtime/debug/stack.go:24 +0x64
main.(*program).run.func1()
        /root/go/aliyun_assist_client/rootcmd.go:301 +0x34
panic({0xa3e120, 0x14f9ae0})
        /root/.g/versions/1.20.5/src/runtime/panic.go:884 +0x1f4
github.com/aliyun/aliyun_assist_client/agent/pluginmanager.InitPluginCheckTimer()
        /root/go/aliyun_assist_client/agent/pluginmanager/pluginmanager.go:64 +0x10c
main.(*program).run(0x0?)
        /root/go/aliyun_assist_client/rootcmd.go:345 +0x8a4
created by main.(*program).Start
        /root/go/aliyun_assist_client/rootcmd.go:254 +0x5c`
	os.WriteFile(mokeLogFile, []byte(logcontent), os.ModePerm)
	defer os.Remove(mokeLogFile)
	modifytime, _ := time.Parse("2006-01-02 15:04:05", mokePanicTime)
	modifytime = modifytime.Local().Add(-time.Hour * 8)
	os.Chtimes(mokeLogFile, modifytime.UTC().Local(), modifytime.UTC().Local())
	guard_getStderrLogPath := gomonkey.ApplyFunc(getStderrLogPath, func() (string, error) { return mokeLogFile, nil })
	defer guard_getStderrLogPath.Reset()
	guard_IsSystemdLinux = gomonkey.ApplyFunc(util.IsSystemdLinux, func() bool { return false })
	defer guard_IsSystemdLinux.Reset()
	reported = false
	err = CheckAgentPanic()
	assert.Equal(t, nil, err)
	assert.Equal(t, true, reported)
}
