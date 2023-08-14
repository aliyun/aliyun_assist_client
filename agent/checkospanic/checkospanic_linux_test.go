package checkospanic

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"bou.ke/monkey"
	"github.com/aliyun/aliyun_assist_client/agent/metrics"
	"github.com/aliyun/aliyun_assist_client/agent/util"
	"github.com/stretchr/testify/assert"
)

type _fakeFileInfo struct{}

func (_fakeFileInfo) Name() string       { return "" }
func (_fakeFileInfo) Size() int64        { return 1 }
func (_fakeFileInfo) Mode() fs.FileMode  { return os.ModePerm }
func (_fakeFileInfo) ModTime() time.Time { return time.Now() }
func (_fakeFileInfo) IsDir() bool        { return true }
func (_fakeFileInfo) Sys() interface{}   { return "" }

type _fakeDirEntry struct {}
func (_fakeDirEntry) Name() string { return "" }
func (_fakeDirEntry) IsDir() bool                { return true }
func (_fakeDirEntry) Type() fs.FileMode          { return os.ModePerm }
func (_fakeDirEntry) Info() (fs.FileInfo, error) { return _fakeFileInfo{}, nil }

type _fakeDirEntryA struct {
	_fakeDirEntry
}
func (_fakeDirEntryA) Name() string {
	return "web-server2-kernel-" + time.Now().Format("2006-01-02-15:04:05")
}
type _fakeDirEntryB struct {
	_fakeDirEntry
}
func (_fakeDirEntryB) Name() string {
	return "127.0.0.1-" + time.Now().Format("2006-01-02-15:04:05")
}

func TestReportLastOsPanic(t *testing.T) {
	testFileList := getTestFileList()
	guard_1 := monkey.Patch(util.CheckFileIsExist, func(filename string) bool { return true })
	defer guard_1.Unpatch()
	guard_2 := monkey.Patch(os.ReadFile, func(name string) ([]byte, error) {
		var content string
		if name == "kdumpConfigPath" {
			content = `#ext4 LABEL=/boot
#ext4 UUID=03138356-5e61-4ab3-b58e-27507ac41937
#nfs my.server.com:/export/tmp
#ssh user@my.server.com
#sshkey /root/.ssh/kdump_id_rsa
path /var/crash
core_collector makedumpfile -l --message-level 1 -d 31
#core_collector scp
#kdump_post /var/crash/scripts/kdump-post.sh
kdump_pre /var/crash/scripts/kdump-pre.sh
extra_bins /usr/bin/sh`
		} else {
			content = `[79503.417524] sysrq: Trigger a crash
[79503.417951] Kernel panic - not syncing: sysrq triggered crash
[79503.418514] CPU: 2 PID: 6768 Comm: bash Kdump: loaded Tainted: G           OE     5.10.112-11.al8.x86_64 #1
[79503.419421] Hardware name: Alibaba Cloud Alibaba Cloud ECS, BIOS 449e491 04/01/2014
[79503.420125] Call Trace:
[79503.420374]  dump_stack+0x57/0x6a
[79503.420686]  panic+0x10d/0x2e9
[79503.420972]  sysrq_handle_crash+0x16/0x20
[79503.421342]  __handle_sysrq.cold+0x43/0x113
[79503.421730]  write_sysrq_trigger+0x24/0x40
[79503.422106]  proc_reg_write+0x51/0x90
[79503.422447]  vfs_write+0xc1/0x260
[79503.422755]  ksys_write+0x4f/0xc0
[79503.423075]  do_syscall_64+0x33/0x40
[79503.423406]  entry_SYSCALL_64_after_hwframe+0x44/0xa9
[79503.423870] RIP: 0033:0x7fba3a9e7467
[79503.424200] Code: 0d 00 f7 d8 64 89 02 48 c7 c0 ff ff ff ff eb b7 0f 1f 00 f3 0f 1e fa 64 8b 04 25 18 00 00 00 85 c0 75 10 b8 01 00 00 00 0f 05 <48> 3d 00 f0 ff ff 77 51 c3 48 83 ec 28 48 89 54 24 18 48 89 74 24
[79503.425867] RSP: 002b:00007ffe8a38b768 EFLAGS: 00000246 ORIG_RAX: 0000000000000001`
		}
		return []byte(content), nil
	})
	defer guard_2.Unpatch()
	guard_5 := monkey.Patch(metrics.GetLinuxGuestOSPanicEvent, func(keywords ...string) *metrics.MetricsEvent {
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
	guard_3 := monkey.Patch(os.ReadDir, func(name string) ([]os.DirEntry, error) {
		res := []os.DirEntry{_fakeDirEntryB{}, _fakeDirEntryA{}}
		for _, ent := range res {
			currentTimeStr := time.Now().Format("2006-01-02-15:04:05")
			name := ent.Name()
			assert.Equal(t, true, vmcorePathRegex.MatchString(name))
			items := vmcorePathRegex.FindStringSubmatch(name)
			assert.Equal(t, 2, len(items))
			_, err := time.Parse("2006-01-02-15:04:05", items[1])
			assert.Equal(t, nil, err)
			assert.Equal(t, currentTimeStr, items[1])
		}
		return res, nil
	})
	defer guard_3.Unpatch()
	var m *metrics.MetricsEvent
	guard_4 := monkey.PatchInstanceMethod(reflect.TypeOf(m), "ReportEvent", func(*metrics.MetricsEvent) {})
	defer guard_4.Unpatch()
	ReportLastOsPanic()
	// If there are test file for ut
	if len(testFileList) > 0 {
		guard_2.Unpatch()
		for _, testFile := range testFileList {
			guard_6 := monkey.Patch(filepath.Join, func(elem ...string) string {
				return testFile
			})
			fmt.Println(testFile)
			ReportLastOsPanic()
			guard_6.Unpatch()
		}
	}
}

func getTestFileList() []string {
	testFileList := []string{}
	utcaseDir := "/root/golang/aliyun_assist_client/agent/checkospanic/utcase"
	entries, err := os.ReadDir(utcaseDir)
	if err != nil {
		return testFileList
	} else {
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			testFileList = append(testFileList, filepath.Join(utcaseDir, entry.Name()))
		}
	}
	return testFileList
}
