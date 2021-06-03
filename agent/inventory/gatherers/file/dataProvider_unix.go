// +build darwin freebsd linux netbsd openbsd

package file

import (
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/aliyun/aliyun_assist_client/agent/log"

	"github.com/aliyun/aliyun_assist_client/agent/inventory/model"
)

func expand(str string, mapping func(string) string) (newStr string, err error) {
	newStr = os.Expand(str, mapping)
	return
}

//getMetaData gets metadata for the specified file paths
func getMetaData(paths []string) (fileInfo []model.FileData, err error) {
	for _, p := range paths {
		fi, err := os.Stat(p)
		if err != nil {
			log.GetLogger().Error(err)
		} else {
			var data model.FileData
			data.Size = strconv.FormatInt(fi.Size(), 10)
			data.Name = fi.Name()
			data.ModificationTime = fi.ModTime().Format(time.RFC3339)
			data.InstalledDir = filepath.Dir(p)
			fileInfo = append(fileInfo, data)
		}
	}
	return
}
