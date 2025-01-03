package apiserver

import (
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/aliyun/aliyun_assist_client/thirdparty/sirupsen/logrus"

	"github.com/aliyun/aliyun_assist_client/common/pathutil"
	"github.com/aliyun/aliyun_assist_client/common/requester"
)

type RegionIdFileProvider struct {}

func (*RegionIdFileProvider) Name() string {
	return "RegionIdFileProvider"
}

func (p *RegionIdFileProvider) RegionId(logger logrus.FieldLogger) (string, error) {
	regionId := p.getRegionIdInFile()
	if regionId == "" {
		return "", requester.ErrNotProvided
	}
	return regionId, nil
}

func (*RegionIdFileProvider) getRegionIdInFile() string {
	currentVersionDir, _ := pathutil.GetCurrentPath()
	path := filepath.Join(filepath.Dir(currentVersionDir), "region-id")

	if regionIdFile, err := os.Open(path); err == nil {
		if raw, err2 := io.ReadAll(regionIdFile); err2 == nil {
			return strings.TrimSpace(strings.Trim(string(raw), "\r\t\n"))
		}
	}
	return ""
}

func (*RegionIdFileProvider) SaveRegionId(logger logrus.FieldLogger, regionId string) {
	currentVersionDir, _ := pathutil.GetCurrentPath()
	path := filepath.Join(filepath.Dir(currentVersionDir), "region-id")

	err := os.WriteFile(path, []byte(regionId), os.FileMode(0o644))
	if err != nil {
		logger.WithError(err).Warning("Failed to save detected region ID into cache file")
	} else {
		logger.Info("Saved detected region ID into cache file")
	}
}
