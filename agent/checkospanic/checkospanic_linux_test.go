package checkospanic

import (
	"fmt"
	"io/fs"
	"os"
	"reflect"
	"testing"
	"time"

	gomonkey "github.com/agiledragon/gomonkey/v2"
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
	guard_1 := gomonkey.ApplyFunc(util.CheckFileIsExist, func(filename string) bool { return true })
	defer guard_1.Reset()
	contentList := []string{`[79503.417524] sysrq: Trigger a crash
[79503.417951] Kernel panic - not syncing: sysrq triggered crash
[79503.418514] CPU: 2 PID: 6768 Comm: bash Kdump: loaded Tainted: G           OE     5.10.112-11.al8.x86_64 #1
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
[79503.425867] RSP: 002b:00007ffe8a38b768 EFLAGS: 00000246 ORIG_RAX: 0000000000000001`, `2023-08-10 17:25:15.098+0000Booting from 0000:7c00
2023-08-10 17:25:15.100+0000[    0.358003] Kernel panic - not syncing: VFS: Unable to mount root fs on unknown-block(0,0)
2023-08-10 17:25:17.250+0000[    0.358698] CPU: 0 PID: 1 Comm: swapper/0 Not tainted 4.19.91-26.al7.x86_64 #1
2023-08-10 17:25:17.251+0000[    0.359871] Call Trace:
2023-08-10 17:25:17.251+0000[    0.360100]  dump_stack+0x66/0x8b
2023-08-10 17:25:17.251+0000[    0.360414]  panic+0xfa/0x26b
2023-08-10 17:25:17.252+0000[    0.360677]  ? printk+0x48/0x4a
2023-08-10 17:25:17.252+0000[    0.360952]  mount_block_root+0x205/0x2f0
2023-08-10 17:25:17.252+0000[    0.361288]  prepare_namespace+0x13d/0x173
2023-08-10 17:25:17.253+0000[    0.361893]  kernel_init_freeable+0x280/0x2d1
2023-08-10 17:25:17.253+0000[    0.362435]  ? do_early_param+0x89/0x89
2023-08-10 17:25:17.254+0000[    0.362937]  ? rest_init+0xb0/0xb0
2023-08-10 17:25:17.254+0000[    0.363408]  kernel_init+0xa/0x120
2023-08-10 17:25:17.255+0000[    0.363878]  ret_from_fork+0x35/0x40
2023-08-10 17:25:17.255+0000[    0.365378] Kernel Offset: 0x16000000 from 0xffffffff81000000 (relocation range: 0xffffffff80000000-0xffffffffbfffffff)
2023-08-10 17:25:17.257+0000[    0.366547] Rebooting in 1 seconds..`, `2023-08-14 15:53:30.315+0000Kernel panic - not syncing: Out of memory and no killable processes...
2023-08-14 15:53:30.316+0000
2023-08-14 15:53:30.316+0000Pid: 1485, comm: java Not tainted 2.6.32-642.11.1.el6.x86_64 #1
2023-08-14 15:53:30.317+0000Call Trace:
2023-08-14 15:53:30.317+0000 [<ffffffff815482b1>] ? panic+0xa7/0x179
2023-08-14 15:53:30.318+0000 [<ffffffff8113149d>] ? dump_header+0x10d/0x1b0
2023-08-14 15:53:30.319+0000 [<ffffffff81131e4f>] ? out_of_memory+0x38f/0x3c0
2023-08-14 15:53:30.319+0000 [<ffffffff8113e6bc>] ? __alloc_pages_nodemask+0x93c/0x950
2023-08-14 15:53:30.320+0000 [<ffffffff8117794a>] ? alloc_pages_current+0xaa/0x110
2023-08-14 15:53:30.321+0000 [<ffffffff8112e817>] ? __page_cache_alloc+0x87/0x90
2023-08-14 15:53:30.322+0000 [<ffffffff8112e1fe>] ? find_get_page+0x1e/0xa0
2023-08-14 15:53:30.323+0000 [<ffffffff8112f7b7>] ? filemap_fault+0x1a7/0x500
2023-08-14 15:53:30.323+0000 [<ffffffff811591b4>] ? __do_fault+0x54/0x530
2023-08-14 15:53:30.324+0000 [<ffffffff81159787>] ? handle_pte_fault+0xf7/0xb20
2023-08-14 15:53:30.325+0000 [<ffffffff810abb84>] ? hrtimer_start_range_ns+0x14/0x20
2023-08-14 15:53:30.326+0000 [<ffffffff8115a449>] ? handle_mm_fault+0x299/0x3d0
2023-08-14 15:53:30.327+0000 [<ffffffff81052156>] ? __do_page_fault+0x146/0x500
2023-08-14 15:53:30.327+0000 [<ffffffff81046f28>] ? pvclock_clocksource_read+0x58/0xd0
2023-08-14 15:53:30.328+0000 [<ffffffff81045fbc>] ? kvm_clock_read+0x1c/0x20
2023-08-14 15:53:30.329+0000 [<ffffffff81045fc9>] ? kvm_clock_get_cycles+0x9/0x10
2023-08-14 15:53:30.330+0000 [<ffffffff810b1948>] ? getnstimeofday+0x58/0xf0
2023-08-14 15:53:30.330+0000 [<ffffffff8154f09e>] ? do_page_fault+0x3e/0xa0
2023-08-14 15:53:30.331+0000 [<ffffffff8154c3a5>] ? page_fault+0x25/0x30
2023-08-14 15:53:30.332+0000Initializing cgroup subsys cpuset`}
	var contentIdx int
	guard_2 := gomonkey.ApplyFunc(os.ReadFile, func(name string) ([]byte, error) {
		if name == "kdumpConfigPath" {
			return []byte(`#ext4 LABEL=/boot
#ext4 UUID=03138356-5e61-4ab3-b58e-27507ac41937
#nfs my.server.com:/export/tmp
#ssh user@my.server.com
#sshkey /root/.ssh/kdump_id_rsa
path /var/crash
core_collector makedumpfile -l --message-level 1 -d 31
#core_collector scp
#kdump_post /var/crash/scripts/kdump-post.sh
kdump_pre /var/crash/scripts/kdump-pre.sh
extra_bins /usr/bin/sh`), nil
		}
		return []byte(contentList[contentIdx]), nil
	})
	defer guard_2.Reset()
	guard_5 := gomonkey.ApplyFunc(metrics.GetLinuxGuestOSPanicEvent, func(keywords ...string) *metrics.MetricsEvent {
		event := &metrics.MetricsEvent{}
		for i := 0; i+1 < len(keywords); i += 2 {
			if keywords[i] == "rip" {
				continue
			} else {
				fmt.Println(keywords[i], keywords[i+1])
			}
			assert.NotEqual(t, len(keywords[i+1]), 0)
		}
		return event
	})
	defer guard_5.Reset()
	guard_3 := gomonkey.ApplyFunc(os.ReadDir, func(name string) ([]os.DirEntry, error) {
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
	defer guard_3.Reset()
	var m *metrics.MetricsEvent
	guard_4 := gomonkey.ApplyMethod(reflect.TypeOf(m), "ReportEvent", func(*metrics.MetricsEvent) {})
	defer guard_4.Reset()
	for idx := range contentList {
		contentIdx = idx
		ReportLastOsPanic()
	}
}
