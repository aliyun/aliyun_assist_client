package checkagentpanic

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	gomonkey "github.com/agiledragon/gomonkey/v2"
	"github.com/aliyun/aliyun_assist_client/agent/log"
	"github.com/aliyun/aliyun_assist_client/agent/util"
	"github.com/aliyun/aliyun_assist_client/common/pathutil"
	"github.com/stretchr/testify/assert"
)

var (
	_checkPointLog       string
	_panicLog            string
	_checkPointTimestamp int64 // second
	_panicTimestamp      int64
)

func TestSearchPanicInfoFromJournalctl(t *testing.T) {
	guard_IsSystemdLinux := gomonkey.ApplyFunc(util.IsSystemdLinux, func() bool { return true })
	defer guard_IsSystemdLinux.Reset()

	guard_ExecCmd := gomonkey.ApplyFunc(util.ExeCmdWithContext, func(ctx context.Context, cmd string) (error, string, string) {
		// cmd: "journalctl -u aliyun --no-pager --quiet --output=json --since \"2023-12-14 16:25:01\" --reverse | grep \"Agent check point\" | grep \"aliyun-service\" -m 1"
		sinceTime := parseSinceTimeFromCmd(cmd, "since", "--reverse")
		fmt.Println(sinceTime.Unix(), _checkPointTimestamp)
		if sinceTime.Unix() <= _checkPointTimestamp {
			return nil, _checkPointLog, ""
		}
		return nil, "", ""
	})
	defer guard_ExecCmd.Reset()

	guard_execCmd := gomonkey.ApplyFunc(execCmd, func(ctx context.Context, cmd string) (io.ReadCloser, error) {
		// cmd: journalctl -u aliyun --no-pager --quiet --output=json --since \"2023-12-14 14:40:32\" | grep \"aliyun-service\"
		r, w, _ := os.Pipe()
		go func() {
			sinceTime := parseSinceTimeFromCmd(cmd, "since", "|")
			if sinceTime.Unix() <= _panicTimestamp {
				w.WriteString(_panicLog)
			}
			w.Close()
		}()
		return r, nil
	})
	defer guard_execCmd.Reset()

	tests := []struct {
		checkPointTime time.Time
		panicTime      time.Time
		findPanicInfo  bool
	}{
		{
			checkPointTime: time.Now().Add(journalctlTimeLimit - time.Hour),
			panicTime:      time.Now().Add(journalctlTimeLimit - time.Hour),
			findPanicInfo:  false,
		},
		{
			checkPointTime: time.Now().Add(journalctlTimeLimit - time.Hour),
			panicTime:      time.Now().Add(journalctlTimeLimit + time.Hour),
			findPanicInfo:  true,
		},
		{
			checkPointTime: time.Now().Add(journalctlTimeLimit + time.Hour),
			panicTime:      time.Now().Add(journalctlTimeLimit + time.Hour),
			findPanicInfo:  true,
		},
	}
	for idx, tt := range tests {
		fmt.Println("--------------------------- ", idx)
		genJournalctlLog(tt.checkPointTime, tt.panicTime)
		fmt.Printf("checkpointTime: %d %s\n", tt.checkPointTime.Unix(), tt.checkPointTime.Format(timeFormat))
		fmt.Printf("panicTime: %d %s\n", tt.panicTime.Unix(), tt.panicTime.Format(timeFormat))
		panicTime, panicInfo := searchPanicInfoFromJournalctl(log.GetLogger())
		if tt.findPanicInfo != (len(panicInfo) > 0) {
			fmt.Println(idx, "break")
		}
		fmt.Println(panicInfo)
		assert.Equal(t, tt.findPanicInfo, len(panicInfo) > 0)
		if tt.findPanicInfo {
			assert.Equal(t, panicTime.UnixMicro(), tt.panicTime.UnixMicro())
		}
	}
}

func genJournalctlLog(checkPointTime, panicTime time.Time) {
	checkPointLog := `{"SYSLOG_FACILITY":"3","PRIORITY":"6","_MACHINE_ID":"263ecb5f3a73425da7e3b92798e9b155","_PID":"13044","SYSLOG_IDENTIFIER":"aliyun-service","_COMM":"aliyun-service","_TRANSPORT":"stdout","_CMDLINE":"/usr/local/share/aliyun-assist/2.4.3.421/aliyun-service","MESSAGE":"Agent check point","__CURSOR":"s=12a775a969ac4a54b7ad6b7a6f7fcc94;i=5adc9;b=95bc8a874d734fe08ebe334910033bbc;m=44274171;t=601f02612a06d;x=fc33f471bc67b44b","_BOOT_ID":"95bc8a874d734fe08ebe334910033bbc","_SYSTEMD_UNIT":"aliyun.service","_SYSTEMD_SLICE":"system.slice","__MONOTONIC_TIMESTAMP":"1143423345","_GID":"0","_EXE":"/usr/local/share/aliyun-assist/2.4.3.421/aliyun-service","_SYSTEMD_CGROUP":"/system.slice/aliyun.service","_STREAM_ID":"7eda98c07a024491846ecc92eacfe757","_SELINUX_CONTEXT":"unconfined\n","_UID":"0","_HOSTNAME":"ubuntu22","_CAP_EFFECTIVE":"1ffffffffff","_SYSTEMD_INVOCATION_ID":"05de3634af0642b7aa8f4ffc4dab59d9","__REALTIME_TIMESTAMP":"{__REALTIME_TIMESTAMP}"}`
	_checkPointLog = strings.ReplaceAll(checkPointLog, "{__REALTIME_TIMESTAMP}", fmt.Sprint(checkPointTime.UnixMicro()))
	panicLog := `{"__CURSOR":"s=12a775a969ac4a54b7ad6b7a6f7fcc94;i=5adc9;b=95bc8a874d734fe08ebe334910033bbc;m=44274171;t=601f02612a06d;x=fc33f471bc67b44b","_SYSTEMD_CGROUP":"/system.slice/aliyun.service","_EXE":"/usr/local/share/aliyun-assist/2.4.3.421/aliyun-service","_SYSTEMD_SLICE":"system.slice","_HOSTNAME":"ubuntu22","_GID":"0","_CAP_EFFECTIVE":"1ffffffffff","MESSAGE":"Agent check point","SYSLOG_FACILITY":"3","_TRANSPORT":"stdout","_STREAM_ID":"7eda98c07a024491846ecc92eacfe757","_PID":"13044","_SYSTEMD_INVOCATION_ID":"05de3634af0642b7aa8f4ffc4dab59d9","PRIORITY":"6","_COMM":"aliyun-service","_CMDLINE":"/usr/local/share/aliyun-assist/2.4.3.421/aliyun-service","_MACHINE_ID":"263ecb5f3a73425da7e3b92798e9b155","_UID":"0","_SELINUX_CONTEXT":"unconfined\n","_BOOT_ID":"95bc8a874d734fe08ebe334910033bbc","SYSLOG_IDENTIFIER":"aliyun-service","__REALTIME_TIMESTAMP":"{__REALTIME_TIMESTAMP}","_SYSTEMD_UNIT":"aliyun.service","__MONOTONIC_TIMESTAMP":"1143423345"}
{"__CURSOR":"s=12a775a969ac4a54b7ad6b7a6f7fcc94;i=5adca;b=95bc8a874d734fe08ebe334910033bbc;m=447a0daf;t=601f026656ca0;x=163880f50e711224","__REALTIME_TIMESTAMP":"{__REALTIME_TIMESTAMP}","_PID":"13044","_SYSTEMD_INVOCATION_ID":"05de3634af0642b7aa8f4ffc4dab59d9","_HOSTNAME":"ubuntu22","SYSLOG_FACILITY":"3","MESSAGE":"panic: runtime error: invalid memory address or nil pointer dereference","_SELINUX_CONTEXT":"unconfined\n","_EXE":"/usr/local/share/aliyun-assist/2.4.3.421/aliyun-service","_SYSTEMD_UNIT":"aliyun.service","_STREAM_ID":"7eda98c07a024491846ecc92eacfe757","_SYSTEMD_SLICE":"system.slice","PRIORITY":"6","__MONOTONIC_TIMESTAMP":"1148849583","_COMM":"aliyun-service","_MACHINE_ID":"263ecb5f3a73425da7e3b92798e9b155","SYSLOG_IDENTIFIER":"aliyun-service","_SYSTEMD_CGROUP":"/system.slice/aliyun.service","_GID":"0","_CMDLINE":"/usr/local/share/aliyun-assist/2.4.3.421/aliyun-service","_CAP_EFFECTIVE":"1ffffffffff","_TRANSPORT":"stdout","_UID":"0","_BOOT_ID":"95bc8a874d734fe08ebe334910033bbc"}
{"_CMDLINE":"/usr/local/share/aliyun-assist/2.4.3.421/aliyun-service","_SYSTEMD_INVOCATION_ID":"05de3634af0642b7aa8f4ffc4dab59d9","_HOSTNAME":"ubuntu22","_SYSTEMD_SLICE":"system.slice","SYSLOG_IDENTIFIER":"aliyun-service","_TRANSPORT":"stdout","_MACHINE_ID":"263ecb5f3a73425da7e3b92798e9b155","_EXE":"/usr/local/share/aliyun-assist/2.4.3.421/aliyun-service","_SYSTEMD_CGROUP":"/system.slice/aliyun.service","_PID":"13044","_UID":"0","MESSAGE":"goroutine 34 [running]:","_COMM":"aliyun-service","PRIORITY":"6","__CURSOR":"s=12a775a969ac4a54b7ad6b7a6f7fcc94;i=5adcb;b=95bc8a874d734fe08ebe334910033bbc;m=447a0daf;t=601f026656ca0;x=a2b0446cd8bcb93a","__MONOTONIC_TIMESTAMP":"1148849583","__REALTIME_TIMESTAMP":"{__REALTIME_TIMESTAMP}","SYSLOG_FACILITY":"3","_STREAM_ID":"7eda98c07a024491846ecc92eacfe757","_SELINUX_CONTEXT":"unconfined\n","_CAP_EFFECTIVE":"1ffffffffff","_BOOT_ID":"95bc8a874d734fe08ebe334910033bbc","_SYSTEMD_UNIT":"aliyun.service","_GID":"0"}
{"SYSLOG_FACILITY":"3","_SYSTEMD_SLICE":"system.slice","_PID":"13044","_SYSTEMD_INVOCATION_ID":"05de3634af0642b7aa8f4ffc4dab59d9","SYSLOG_IDENTIFIER":"aliyun-service","_UID":"0","__MONOTONIC_TIMESTAMP":"1148849583","_CAP_EFFECTIVE":"1ffffffffff","__REALTIME_TIMESTAMP":"{__REALTIME_TIMESTAMP}","_CMDLINE":"/usr/local/share/aliyun-assist/2.4.3.421/aliyun-service","_SELINUX_CONTEXT":"unconfined\n","_BOOT_ID":"95bc8a874d734fe08ebe334910033bbc","_SYSTEMD_CGROUP":"/system.slice/aliyun.service","PRIORITY":"6","_EXE":"/usr/local/share/aliyun-assist/2.4.3.421/aliyun-service","_SYSTEMD_UNIT":"aliyun.service","_STREAM_ID":"7eda98c07a024491846ecc92eacfe757","_HOSTNAME":"ubuntu22","__CURSOR":"s=12a775a969ac4a54b7ad6b7a6f7fcc94;i=5adcc;b=95bc8a874d734fe08ebe334910033bbc;m=447a0daf;t=601f026656ca0;x=c8e4f0ff2fce8d2d","_GID":"0","_MACHINE_ID":"263ecb5f3a73425da7e3b92798e9b155","_TRANSPORT":"stdout","_COMM":"aliyun-service","MESSAGE":"runtime/debug.Stack()"}
{"SYSLOG_FACILITY":"3","MESSAGE":"\t/root/.g/versions/1.20.5/src/runtime/debug/stack.go:24 +0x64","_SYSTEMD_SLICE":"system.slice","_SELINUX_CONTEXT":"unconfined\n","_HOSTNAME":"ubuntu22","PRIORITY":"6","_TRANSPORT":"stdout","_CAP_EFFECTIVE":"1ffffffffff","_COMM":"aliyun-service","_STREAM_ID":"7eda98c07a024491846ecc92eacfe757","_GID":"0","_UID":"0","_EXE":"/usr/local/share/aliyun-assist/2.4.3.421/aliyun-service","_CMDLINE":"/usr/local/share/aliyun-assist/2.4.3.421/aliyun-service","_SYSTEMD_CGROUP":"/system.slice/aliyun.service","_MACHINE_ID":"263ecb5f3a73425da7e3b92798e9b155","_PID":"13044","_SYSTEMD_UNIT":"aliyun.service","_SYSTEMD_INVOCATION_ID":"05de3634af0642b7aa8f4ffc4dab59d9","__MONOTONIC_TIMESTAMP":"1148849583","__REALTIME_TIMESTAMP":"{__REALTIME_TIMESTAMP}","_BOOT_ID":"95bc8a874d734fe08ebe334910033bbc","SYSLOG_IDENTIFIER":"aliyun-service","__CURSOR":"s=12a775a969ac4a54b7ad6b7a6f7fcc94;i=5adcd;b=95bc8a874d734fe08ebe334910033bbc;m=447a0daf;t=601f026656ca0;x=b99db48f0c5fb7d6"}
{"_SYSTEMD_UNIT":"aliyun.service","SYSLOG_IDENTIFIER":"aliyun-service","_COMM":"aliyun-service","SYSLOG_FACILITY":"3","_SYSTEMD_INVOCATION_ID":"05de3634af0642b7aa8f4ffc4dab59d9","_CMDLINE":"/usr/local/share/aliyun-assist/2.4.3.421/aliyun-service","PRIORITY":"6","_HOSTNAME":"ubuntu22","_SELINUX_CONTEXT":"unconfined\n","_GID":"0","_SYSTEMD_SLICE":"system.slice","_UID":"0","_EXE":"/usr/local/share/aliyun-assist/2.4.3.421/aliyun-service","_PID":"13044","_BOOT_ID":"95bc8a874d734fe08ebe334910033bbc","_SYSTEMD_CGROUP":"/system.slice/aliyun.service","MESSAGE":"main.(*program).run.func1()","__REALTIME_TIMESTAMP":"{__REALTIME_TIMESTAMP}","_TRANSPORT":"stdout","__MONOTONIC_TIMESTAMP":"1148849583","_CAP_EFFECTIVE":"1ffffffffff","_MACHINE_ID":"263ecb5f3a73425da7e3b92798e9b155","_STREAM_ID":"7eda98c07a024491846ecc92eacfe757","__CURSOR":"s=12a775a969ac4a54b7ad6b7a6f7fcc94;i=5adce;b=95bc8a874d734fe08ebe334910033bbc;m=447a0daf;t=601f026656ca0;x=94c4c924cb59094"}
{"_CAP_EFFECTIVE":"1ffffffffff","SYSLOG_FACILITY":"3","_SYSTEMD_SLICE":"system.slice","MESSAGE":"\t/root/go/aliyun_assist_client/rootcmd.go:301 +0x34","_PID":"13044","_EXE":"/usr/local/share/aliyun-assist/2.4.3.421/aliyun-service","_SYSTEMD_CGROUP":"/system.slice/aliyun.service","__REALTIME_TIMESTAMP":"{__REALTIME_TIMESTAMP}","_CMDLINE":"/usr/local/share/aliyun-assist/2.4.3.421/aliyun-service","_COMM":"aliyun-service","_SYSTEMD_INVOCATION_ID":"05de3634af0642b7aa8f4ffc4dab59d9","_BOOT_ID":"95bc8a874d734fe08ebe334910033bbc","PRIORITY":"6","_MACHINE_ID":"263ecb5f3a73425da7e3b92798e9b155","_UID":"0","SYSLOG_IDENTIFIER":"aliyun-service","_SYSTEMD_UNIT":"aliyun.service","__MONOTONIC_TIMESTAMP":"1148849583","_STREAM_ID":"7eda98c07a024491846ecc92eacfe757","_HOSTNAME":"ubuntu22","__CURSOR":"s=12a775a969ac4a54b7ad6b7a6f7fcc94;i=5adcf;b=95bc8a874d734fe08ebe334910033bbc;m=447a0daf;t=601f026656ca0;x=de3f022ca63f800d","_TRANSPORT":"stdout","_GID":"0","_SELINUX_CONTEXT":"unconfined\n"}
{"SYSLOG_FACILITY":"3","_STREAM_ID":"7eda98c07a024491846ecc92eacfe757","_COMM":"aliyun-service","__REALTIME_TIMESTAMP":"{__REALTIME_TIMESTAMP}","_SELINUX_CONTEXT":"unconfined\n","_EXE":"/usr/local/share/aliyun-assist/2.4.3.421/aliyun-service","_GID":"0","_SYSTEMD_SLICE":"system.slice","_PID":"13044","_SYSTEMD_INVOCATION_ID":"05de3634af0642b7aa8f4ffc4dab59d9","_MACHINE_ID":"263ecb5f3a73425da7e3b92798e9b155","_TRANSPORT":"stdout","SYSLOG_IDENTIFIER":"aliyun-service","PRIORITY":"6","__CURSOR":"s=12a775a969ac4a54b7ad6b7a6f7fcc94;i=5add0;b=95bc8a874d734fe08ebe334910033bbc;m=447a0daf;t=601f026656ca0;x=9389ee63b31232ca","_CAP_EFFECTIVE":"1ffffffffff","_UID":"0","_SYSTEMD_CGROUP":"/system.slice/aliyun.service","_SYSTEMD_UNIT":"aliyun.service","_HOSTNAME":"ubuntu22","_BOOT_ID":"95bc8a874d734fe08ebe334910033bbc","MESSAGE":"panic({0xa3e120, 0x14f9ae0})","__MONOTONIC_TIMESTAMP":"1148849583","_CMDLINE":"/usr/local/share/aliyun-assist/2.4.3.421/aliyun-service"}
{"_STREAM_ID":"7eda98c07a024491846ecc92eacfe757","_SYSTEMD_INVOCATION_ID":"05de3634af0642b7aa8f4ffc4dab59d9","_SYSTEMD_UNIT":"aliyun.service","_TRANSPORT":"stdout","_PID":"13044","_GID":"0","SYSLOG_FACILITY":"3","_SYSTEMD_SLICE":"system.slice","_SELINUX_CONTEXT":"unconfined\n","_BOOT_ID":"95bc8a874d734fe08ebe334910033bbc","_MACHINE_ID":"263ecb5f3a73425da7e3b92798e9b155","_HOSTNAME":"ubuntu22","_SYSTEMD_CGROUP":"/system.slice/aliyun.service","_EXE":"/usr/local/share/aliyun-assist/2.4.3.421/aliyun-service","_UID":"0","_COMM":"aliyun-service","__REALTIME_TIMESTAMP":"{__REALTIME_TIMESTAMP}","__MONOTONIC_TIMESTAMP":"1148849583","MESSAGE":"\t/root/.g/versions/1.20.5/src/runtime/panic.go:884 +0x1f4","_CMDLINE":"/usr/local/share/aliyun-assist/2.4.3.421/aliyun-service","PRIORITY":"6","_CAP_EFFECTIVE":"1ffffffffff","SYSLOG_IDENTIFIER":"aliyun-service","__CURSOR":"s=12a775a969ac4a54b7ad6b7a6f7fcc94;i=5add1;b=95bc8a874d734fe08ebe334910033bbc;m=447a0daf;t=601f026656ca0;x=46775d46beadb584"}
{"_UID":"0","SYSLOG_IDENTIFIER":"aliyun-service","_MACHINE_ID":"263ecb5f3a73425da7e3b92798e9b155","_BOOT_ID":"95bc8a874d734fe08ebe334910033bbc","_SYSTEMD_SLICE":"system.slice","MESSAGE":"github.com/aliyun/aliyun_assist_client/agent/pluginmanager.InitPluginCheckTimer()","_TRANSPORT":"stdout","_EXE":"/usr/local/share/aliyun-assist/2.4.3.421/aliyun-service","__CURSOR":"s=12a775a969ac4a54b7ad6b7a6f7fcc94;i=5add2;b=95bc8a874d734fe08ebe334910033bbc;m=447a0daf;t=601f026656ca0;x=f0f07ce5df9d2516","__MONOTONIC_TIMESTAMP":"1148849583","_SYSTEMD_INVOCATION_ID":"05de3634af0642b7aa8f4ffc4dab59d9","_COMM":"aliyun-service","_GID":"0","_SYSTEMD_UNIT":"aliyun.service","__REALTIME_TIMESTAMP":"{__REALTIME_TIMESTAMP}","_PID":"13044","_CAP_EFFECTIVE":"1ffffffffff","_SYSTEMD_CGROUP":"/system.slice/aliyun.service","_CMDLINE":"/usr/local/share/aliyun-assist/2.4.3.421/aliyun-service","SYSLOG_FACILITY":"3","_HOSTNAME":"ubuntu22","_STREAM_ID":"7eda98c07a024491846ecc92eacfe757","PRIORITY":"6","_SELINUX_CONTEXT":"unconfined\n"}
{"_SYSTEMD_INVOCATION_ID":"05de3634af0642b7aa8f4ffc4dab59d9","_HOSTNAME":"ubuntu22","SYSLOG_FACILITY":"3","_MACHINE_ID":"263ecb5f3a73425da7e3b92798e9b155","_STREAM_ID":"7eda98c07a024491846ecc92eacfe757","_CMDLINE":"/usr/local/share/aliyun-assist/2.4.3.421/aliyun-service","__MONOTONIC_TIMESTAMP":"1148849583","_TRANSPORT":"stdout","_SYSTEMD_CGROUP":"/system.slice/aliyun.service","_BOOT_ID":"95bc8a874d734fe08ebe334910033bbc","PRIORITY":"6","_SYSTEMD_UNIT":"aliyun.service","MESSAGE":"\t/root/go/aliyun_assist_client/agent/pluginmanager/pluginmanager.go:64 +0x10c","_CAP_EFFECTIVE":"1ffffffffff","_SELINUX_CONTEXT":"unconfined\n","__CURSOR":"s=12a775a969ac4a54b7ad6b7a6f7fcc94;i=5add3;b=95bc8a874d734fe08ebe334910033bbc;m=447a0daf;t=601f026656ca0;x=c6a211eb61a74fa","__REALTIME_TIMESTAMP":"{__REALTIME_TIMESTAMP}","_UID":"0","_SYSTEMD_SLICE":"system.slice","SYSLOG_IDENTIFIER":"aliyun-service","_GID":"0","_PID":"13044","_EXE":"/usr/local/share/aliyun-assist/2.4.3.421/aliyun-service","_COMM":"aliyun-service"}
{"__MONOTONIC_TIMESTAMP":"1148849583","_HOSTNAME":"ubuntu22","_MACHINE_ID":"263ecb5f3a73425da7e3b92798e9b155","_UID":"0","_COMM":"aliyun-service","__REALTIME_TIMESTAMP":"{__REALTIME_TIMESTAMP}","SYSLOG_FACILITY":"3","_SYSTEMD_SLICE":"system.slice","_SYSTEMD_CGROUP":"/system.slice/aliyun.service","_GID":"0","_SELINUX_CONTEXT":"unconfined\n","_CAP_EFFECTIVE":"1ffffffffff","__CURSOR":"s=12a775a969ac4a54b7ad6b7a6f7fcc94;i=5add4;b=95bc8a874d734fe08ebe334910033bbc;m=447a0daf;t=601f026656ca0;x=d8b568749285f949","_SYSTEMD_INVOCATION_ID":"05de3634af0642b7aa8f4ffc4dab59d9","PRIORITY":"6","_EXE":"/usr/local/share/aliyun-assist/2.4.3.421/aliyun-service","_CMDLINE":"/usr/local/share/aliyun-assist/2.4.3.421/aliyun-service","_BOOT_ID":"95bc8a874d734fe08ebe334910033bbc","_PID":"13044","_SYSTEMD_UNIT":"aliyun.service","_STREAM_ID":"7eda98c07a024491846ecc92eacfe757","MESSAGE":"main.(*program).run(0x0?)","SYSLOG_IDENTIFIER":"aliyun-service","_TRANSPORT":"stdout"}
{"SYSLOG_FACILITY":"3","PRIORITY":"6","__REALTIME_TIMESTAMP":"{__REALTIME_TIMESTAMP}","_SYSTEMD_UNIT":"aliyun.service","_STREAM_ID":"7eda98c07a024491846ecc92eacfe757","_SYSTEMD_SLICE":"system.slice","_CMDLINE":"/usr/local/share/aliyun-assist/2.4.3.421/aliyun-service","__CURSOR":"s=12a775a969ac4a54b7ad6b7a6f7fcc94;i=5add5;b=95bc8a874d734fe08ebe334910033bbc;m=447a0daf;t=601f026656ca0;x=4a15c9443bd443f8","_HOSTNAME":"ubuntu22","SYSLOG_IDENTIFIER":"aliyun-service","_SYSTEMD_CGROUP":"/system.slice/aliyun.service","_EXE":"/usr/local/share/aliyun-assist/2.4.3.421/aliyun-service","_SELINUX_CONTEXT":"unconfined\n","_UID":"0","_PID":"13044","_TRANSPORT":"stdout","_CAP_EFFECTIVE":"1ffffffffff","__MONOTONIC_TIMESTAMP":"1148849583","_MACHINE_ID":"263ecb5f3a73425da7e3b92798e9b155","MESSAGE":"\t/root/go/aliyun_assist_client/rootcmd.go:345 +0x8a4","_COMM":"aliyun-service","_BOOT_ID":"95bc8a874d734fe08ebe334910033bbc","_GID":"0","_SYSTEMD_INVOCATION_ID":"05de3634af0642b7aa8f4ffc4dab59d9"}
{"SYSLOG_IDENTIFIER":"aliyun-service","_COMM":"aliyun-service","_CMDLINE":"/usr/local/share/aliyun-assist/2.4.3.421/aliyun-service","_BOOT_ID":"95bc8a874d734fe08ebe334910033bbc","_SYSTEMD_SLICE":"system.slice","_SELINUX_CONTEXT":"unconfined\n","_SYSTEMD_UNIT":"aliyun.service","__CURSOR":"s=12a775a969ac4a54b7ad6b7a6f7fcc94;i=5add6;b=95bc8a874d734fe08ebe334910033bbc;m=447a0daf;t=601f026656ca0;x=76742e76725d70b1","PRIORITY":"6","_GID":"0","_SYSTEMD_INVOCATION_ID":"05de3634af0642b7aa8f4ffc4dab59d9","_STREAM_ID":"7eda98c07a024491846ecc92eacfe757","_MACHINE_ID":"263ecb5f3a73425da7e3b92798e9b155","_SYSTEMD_CGROUP":"/system.slice/aliyun.service","_CAP_EFFECTIVE":"1ffffffffff","MESSAGE":"created by main.(*program).Start","_TRANSPORT":"stdout","_HOSTNAME":"ubuntu22","_UID":"0","_EXE":"/usr/local/share/aliyun-assist/2.4.3.421/aliyun-service","_PID":"13044","SYSLOG_FACILITY":"3","__MONOTONIC_TIMESTAMP":"1148849583","__REALTIME_TIMESTAMP":"{__REALTIME_TIMESTAMP}"}
{"__REALTIME_TIMESTAMP":"{__REALTIME_TIMESTAMP}","PRIORITY":"6","_TRANSPORT":"stdout","SYSLOG_FACILITY":"3","_GID":"0","_EXE":"/usr/local/share/aliyun-assist/2.4.3.421/aliyun-service","_CAP_EFFECTIVE":"1ffffffffff","_UID":"0","__CURSOR":"s=12a775a969ac4a54b7ad6b7a6f7fcc94;i=5add7;b=95bc8a874d734fe08ebe334910033bbc;m=447a0daf;t=601f026656ca0;x=98a083a8161a2d8b","_HOSTNAME":"ubuntu22","_CMDLINE":"/usr/local/share/aliyun-assist/2.4.3.421/aliyun-service","_SYSTEMD_SLICE":"system.slice","_BOOT_ID":"95bc8a874d734fe08ebe334910033bbc","__MONOTONIC_TIMESTAMP":"1148849583","MESSAGE":"\t/root/go/aliyun_assist_client/rootcmd.go:254 +0x5c","_STREAM_ID":"7eda98c07a024491846ecc92eacfe757","_COMM":"aliyun-service","_SYSTEMD_INVOCATION_ID":"05de3634af0642b7aa8f4ffc4dab59d9","SYSLOG_IDENTIFIER":"aliyun-service","_MACHINE_ID":"263ecb5f3a73425da7e3b92798e9b155","_SYSTEMD_UNIT":"aliyun.service","_SELINUX_CONTEXT":"unconfined\n","_PID":"13044","_SYSTEMD_CGROUP":"/system.slice/aliyun.service"}`
	_panicLog = strings.ReplaceAll(panicLog, "{__REALTIME_TIMESTAMP}", fmt.Sprint(panicTime.UnixMicro()))
	_panicTimestamp = panicTime.Unix()
	_checkPointTimestamp = checkPointTime.Unix()
}

func parseSinceTimeFromCmd(cmd, leftStr, rightStr string) time.Time {
	s := strings.Index(cmd, leftStr) + len(leftStr)
	e := strings.Index(cmd, rightStr)
	if s < 0 || e < 0 || s >= e {
		return time.Time{}
	}
	sinceTimeStr := strings.Trim(cmd[s:e], "\" ")
	location, _ := time.LoadLocation("Asia/Shanghai")
	sinceTime, _ := time.ParseInLocation(timeFormat, sinceTimeStr, location)
	return sinceTime
}

func TestRedirectStdouterr(t *testing.T) {
	stdouterrDir, _ := pathutil.GetLogPath()
	stdoutFile := filepath.Join(stdouterrDir, stdoutFileName)
	stderrFile := filepath.Join(stdouterrDir, stderrFileName)
	defer func() {
		os.RemoveAll(stdouterrDir)
	}()

	guard_isSystemd := gomonkey.ApplyFunc(util.IsSystemdLinux, func() bool {
		return false
	})
	defer guard_isSystemd.Reset()

	fmt.Fprintf(os.Stdout, "stdout: should not record")
	fmt.Fprintf(os.Stderr, "stderr: should not record")

	RedirectStdouterr()

	stdoutContent := `stdout: should record 1
stdout: should record 2`
	stderrContent := `stderr: should record 1
stderr: should record 2`
	fmt.Fprint(os.Stdout, stdoutContent)
	fmt.Fprint(os.Stderr, stderrContent)

	stdoutFileContent, _ := os.ReadFile(stdoutFile)
	stderrFileContent, _ := os.ReadFile(stderrFile)
	assert.Equal(t, stdoutContent, string(stdoutFileContent))
	assert.Equal(t, stderrContent, string(stderrFileContent))
}

func TestExecCmd(t *testing.T) {
	tests := []struct {
		cmd            string
		timeout        int
		expectedStdout string
	}{
		{
			cmd:            "echo 111",
			timeout:        1,
			expectedStdout: "111",
		},
		{
			cmd:            "echo 111; sleep 2; echo 222",
			timeout:        1,
			expectedStdout: "111",
		},
		{
			cmd:            "echo 111; sleep 1; echo 222",
			timeout:        2,
			expectedStdout: "111222",
		},
	}
	for _, tt := range tests {
		func() {
			startTime := time.Now()

			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(tt.timeout)*time.Second)
			defer cancel()
			r, err := execCmd(ctx, tt.cmd)
			assert.Equal(t, nil, err)

			scanner := bufio.NewScanner(r)
			scanner.Split(bufio.ScanLines)
			output := []byte{}
			for scanner.Scan() {
				output = append(output, scanner.Bytes()...)
			}
			assert.Equal(t, tt.expectedStdout, string(output))
			assert.True(t, time.Since(startTime) < time.Duration(tt.timeout)*time.Second+time.Millisecond*time.Duration(10))
		}()
	}
}
