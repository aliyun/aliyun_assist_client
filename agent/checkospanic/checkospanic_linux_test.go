package checkospanic

import (
	"bytes"
	"compress/flate"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	gomonkey "github.com/agiledragon/gomonkey/v2"
	"github.com/aliyun/aliyun_assist_client/agent/log"
	"github.com/aliyun/aliyun_assist_client/agent/metrics"
	"github.com/aliyun/aliyun_assist_client/common/fileutil"
	"github.com/stretchr/testify/assert"
)

type _fakeFileInfo struct{}

func (_fakeFileInfo) Name() string       { return "" }
func (_fakeFileInfo) Size() int64        { return 1 }
func (_fakeFileInfo) Mode() fs.FileMode  { return os.ModePerm }
func (_fakeFileInfo) ModTime() time.Time { return time.Now() }
func (_fakeFileInfo) IsDir() bool        { return true }
func (_fakeFileInfo) Sys() interface{}   { return "" }

type _fakeDirEntry struct{}

func (_fakeDirEntry) Name() string               { return "" }
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
	guard_1 := gomonkey.ApplyFunc(fileutil.CheckFileIsExist, func(filename string) bool { return true })
	defer guard_1.Reset()
	
	guard_2 := gomonkey.ApplyFunc(os.ReadFile, func(name string) ([]byte, error) {
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
	})
	defer guard_2.Reset()

	guard_5 := gomonkey.ApplyFunc(metrics.GetLinuxGuestOSPanicEvent, func(keywords ...string) *metrics.MetricsEvent {
		event := &metrics.MetricsEvent{
			KeyWords: genKeyWordsStr(keywords...),
		}
		for i := 0; i+1 < len(keywords); i += 2 {
			fmt.Println(keywords[i], keywords[i+1])
			if keywords[i] == "rip" || keywords[i] == "rawContent" || keywords[i] == "errMsg" {
				continue
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
			_, err := time.ParseInLocation("2006-01-02-15:04:05", items[1], time.Local)
			assert.Equal(t, nil, err)
			assert.Equal(t, currentTimeStr, items[1])
		}
		return res, nil
	})
	defer guard_3.Reset()

	var m *metrics.MetricsEvent
	guard_4 := gomonkey.ApplyMethod(reflect.TypeOf(m), "ReportEvent", func(me *metrics.MetricsEvent) {
		keywords := make(map[string]string)
		err := json.Unmarshal([]byte(me.KeyWords), &keywords)
		assert.Equal(t, nil, err)
		rawContent, ok := keywords["rawContent"]
		assert.Equal(t, true, ok)
		errMsg, ok := keywords["errMsg"]
		assert.Equal(t, true, ok)
		assert.Equal(t, nil, errMsg)
		_, err = decompressFlate(rawContent)
		assert.Equal(t, nil, err)
		
	})
	defer guard_4.Reset()	
}


func TestFindLocalVmcoreDmesg(t *testing.T) {
	guard_1 := gomonkey.ApplyFunc(fileutil.CheckFileIsExist, func(filename string) bool { return true })
	defer guard_1.Reset()
	
	guard_2 := gomonkey.ApplyFunc(os.ReadFile, func(name string) ([]byte, error) {
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
	})
	defer guard_2.Reset()

	guard_3 := gomonkey.ApplyFunc(os.ReadDir, func(name string) ([]os.DirEntry, error) {
		res := []os.DirEntry{_fakeDirEntryB{}, _fakeDirEntryA{}}
		for _, ent := range res {
			currentTimeStr := time.Now().Format("2006-01-02-15:04:05")
			name := ent.Name()
			assert.Equal(t, true, vmcorePathRegex.MatchString(name))
			items := vmcorePathRegex.FindStringSubmatch(name)
			assert.Equal(t, 2, len(items))
			_, err := time.ParseInLocation("2006-01-02-15:04:05", items[1], time.Local)
			assert.Equal(t, nil, err)
			assert.Equal(t, currentTimeStr, items[1])
		}
		return res, nil
	})
	defer guard_3.Reset()

	vmcoreDmesgPath, latestDir, latestTime := FindLocalVmcoreDmesg(log.GetLogger())
	assert.Equal(t, filepath.Join("/var/crash/", _fakeDirEntryB{}.Name(), vmcoreDmesgFile), vmcoreDmesgPath)
	assert.Equal(t, _fakeDirEntryB{}.Name(), latestDir)
	assert.NotEqual(t, "", latestTime)
}

func TestParseVmcore(t *testing.T) {
	root := "testfile" // Replace with the root directory you want to traverse
	fileList := []string{}
	err := filepath.Walk(root, func (path string, info os.FileInfo, err error) error {
		if info != nil && !info.IsDir() {
			fileList = append(fileList, path)	
		}
		return nil
	})
	assert.Equal(t, nil, err)
	for _, testfile := range fileList {
		// fmt.Println(testfile)
		_, _, panicInfo, rawContent, err := ParseVmcore(log.GetLogger(), testfile)
		// if len(panicInfo) == 0 {
		// 	fmt.Printf("-------------------%s-----------------------\n", testfile)
		// }
		assert.Equal(t, nil, err)
		assert.NotEqual(t, nil, panicInfo)
		assert.Equal(t, true, len(strings.Split(rawContent, "\n")) <= maxLinesAfterPanicInfo+maxLinesBeforePanicInfo+1)
		
	}
}

func TestCompressFlate(t *testing.T) {
	rawContent := "hello"
	compressed, err := compressFlate(rawContent)
	assert.Equal(t, nil, err)
	decompressed, err := decompressFlate(compressed)
	assert.Equal(t, nil, err)
	assert.Equal(t, rawContent, decompressed)
}

func decompressFlate(input string) (string, error) {
	if len(input) == 0 {
		return "", nil
	}
	decoded, err := base64.StdEncoding.DecodeString(input)
	if err != nil {
		return "", err
	}
	buf := bytes.NewBuffer(decoded)
	flateReader := flate.NewReader(buf)
	defer flateReader.Close()
	deBuffer := new(bytes.Buffer)
	if _, err := io.Copy(deBuffer, flateReader); err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
		return "", err
	}
	return deBuffer.String(), nil
}

func genKeyWordsStr(keywords ...string) string {
	if len(keywords) >= 2 {
		kmp := make(map[string]string)
		for i := 0; i < len(keywords); i += 2 {
			kmp[keywords[i]] = keywords[i+1]
		}
		kmpStr, _ := json.Marshal(&kmp)
		return string(kmpStr)
	}
	return ""
}
