package container

import (
	"time"

	"github.com/aliyun/aliyun_assist_client/thirdparty/sirupsen/logrus"

	libcri "github.com/aliyun/aliyun_assist_client/plugin/commander/container/util/cri"
	"github.com/aliyun/aliyun_assist_client/plugin/commander/container/util/docker"
	"github.com/aliyun/aliyun_assist_client/plugin/commander/container/util/model"
	"github.com/aliyun/aliyun_assist_client/agent/log"
)

type ListContainersOptions struct {
	ConnectTimeout    time.Duration
	DataSourceName    string
	ShowAllContainers bool
}

func ListContainers(opts ListContainersOptions) ([]model.Container, error) {
	log.GetLogger().WithFields(logrus.Fields{
		"connectTimeout": opts.ConnectTimeout,
		"dataSourceName": opts.DataSourceName,
	}).Infoln("Would retrieve container list from specified data source in limited time")

	var containers []model.Container
	var err error
	if opts.DataSourceName == "all" || opts.DataSourceName == "cri" {
		containersFromCRI, errFromCRI := libcri.ListContainers(opts.ConnectTimeout, opts.ShowAllContainers)
		if errFromCRI != nil {
			log.GetLogger().WithError(errFromCRI).Errorln("Failed to retrieve container list from CRI")
			err = errFromCRI
		}
		if len(containersFromCRI) > 0 {
			containers = append(containers, containersFromCRI...)
		}
	}
	if opts.DataSourceName == "all" || opts.DataSourceName == "docker" {
		containersFromDocker, errFromDocker := docker.ListContainers(opts.ConnectTimeout, opts.ShowAllContainers)
		if errFromDocker != nil {
			log.GetLogger().WithError(errFromDocker).Errorln("Failed to retrieve container list from docker")
			// Keep previous error encountered
			if err == nil {
				err = errFromDocker
			}
		}
		if len(containersFromDocker) > 0 {
			containers = append(containers, containersFromDocker...)
		}
	}

	// Deduplicate containers with the same ID, but retrieved from both CRI and
	// docker
	if len(containers) > 0 {
		containerIdSet := make(map[string]struct{}, len(containers))

		uniqueContainers := make([]model.Container, 0, len(containers))
		for _, container := range containers {
			if _, ok := containerIdSet[container.Id]; !ok {
				containerIdSet[container.Id] = struct{}{}
				uniqueContainers = append(uniqueContainers, container)
			}
		}
		containers = uniqueContainers
	}

	return containers, err
}
