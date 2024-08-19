package checknet

import (
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"testing"

	gomonkey "github.com/agiledragon/gomonkey/v2"
	"github.com/jarcoal/httpmock"

	"github.com/aliyun/aliyun_assist_client/common/fileutil"
	"github.com/aliyun/aliyun_assist_client/agent/util"
	"github.com/aliyun/aliyun_assist_client/common/networkcategory"
)

func deletefile(file string) {
	if fileutil.CheckFileIsExist(file) {
		os.Remove(file)
	}
}

func TestNetWorkCheck(t *testing.T) {
	_needToReport.Set()
	defer _needToReport.Clear()

	networkCategoryCache.Set(networkcategory.NetworkVPC)

	path, _ := os.Executable()
	currentVersionDir, _ := filepath.Abs(filepath.Dir(path))
	currentVersionNetcheckPath := filepath.Join(currentVersionDir, "aliyun_assist_netcheck")
	os.Create(currentVersionNetcheckPath)
	defer deletefile(currentVersionNetcheckPath)
	
	var cmd *exec.Cmd
	guard_1 := gomonkey.ApplyMethod(reflect.TypeOf(cmd), "Run", func(*exec.Cmd) error {
		return nil
	})
	defer guard_1.Reset()

	RequestNetcheck("-")
	RequestNetcheck(NetcheckRequestNormal)	
	RequestNetcheck(NetcheckRequestForceOnce)
	_doNetcheck(NetcheckRequestNormal)
	_doNetcheck(NetcheckRequestForceOnce)
	RecentReport()

	httpmock.Activate()
	util.NilRequest.Set()
	defer util.NilRequest.Clear()
	defer httpmock.DeactivateAndReset()
	url := "http://checknet.c"
	downloadfile := filepath.Join(currentVersionDir, "downloadfile")
	defer deletefile(downloadfile)
	httpmock.RegisterResponder("GET", url, func(h *http.Request) (*http.Response, error) { return httpmock.NewStringResponse(200, "ok"), nil})
	httpmock.RegisterResponder("POST", url, func(h *http.Request) (*http.Response, error) { return httpmock.NewStringResponse(200, "ok"), nil})
	HttpGet(url)
	HttpPost(url, "", "")
}