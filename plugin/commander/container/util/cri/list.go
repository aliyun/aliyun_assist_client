package cri

import (
	"sort"
	"strings"
	"time"

	criapis "k8s.io/cri-api/pkg/apis"
	runtimeapis "k8s.io/cri-api/pkg/apis/runtime/v1"
	"k8s.io/kubernetes/pkg/kubelet/cri/remote"

	"github.com/aliyun/aliyun_assist_client/plugin/commander/container/util/model"
	"github.com/aliyun/aliyun_assist_client/agent/log"
)

func ListContainers(connectTimeout time.Duration, showAllContainers bool) ([]model.Container, error) {
	containerFilter := &runtimeapis.ContainerFilter{}
	if !showAllContainers {
		containerFilter.State = &runtimeapis.ContainerStateValue{
			State: runtimeapis.ContainerState_CONTAINER_RUNNING,
		}
	}

	containers := []model.Container{}
	// Since both old dockershim inside kubelet and new standalone cri-docker
	// provide CRI to docker, runtime endpoints need to be deduplicated based on
	// runtime name.
	uniqueRuntimeNames := make([]string, len(RuntimeName2Endpoints))
	for uniqueRuntimeName := range RuntimeName2Endpoints {
		uniqueRuntimeNames = append(uniqueRuntimeNames, uniqueRuntimeName)
	}
	sort.Strings(uniqueRuntimeNames)

	for _, sortedRuntimeName := range uniqueRuntimeNames {
		for _, endpoint := range RuntimeName2Endpoints[sortedRuntimeName] {
			service, err := remote.NewRemoteRuntimeService(endpoint.Endpoint, connectTimeout)
			if err != nil {
				log.GetLogger().WithError(err).Errorf("Failed to connect %s runtime via %s", endpoint.RuntimeName, endpoint.Endpoint)
				continue
			}

			oneRuntimeContainers, err := listCRIContainers(service, endpoint.RuntimeName, containerFilter)
			if err != nil {
				log.GetLogger().WithError(err).Errorf("Failed to list containers on %s runtime via %s", endpoint.RuntimeName, endpoint.Endpoint)
				continue
			}

			// Sort containers on one runtime by container id lexicographically
			sort.SliceStable(oneRuntimeContainers, func(i, j int) bool {
				return oneRuntimeContainers[i].Id < oneRuntimeContainers[j].Id
			})

			containers = append(containers, oneRuntimeContainers...)
			// Only one available runtime endpoint is needed for the container
			// runtime
			break
		}
	}

	return containers, nil
}

func listCRIContainers(service criapis.RuntimeService, runtimeName string, containerFilter *runtimeapis.ContainerFilter) ([]model.Container, error) {
	criPodSandboxes, err := service.ListPodSandbox(&runtimeapis.PodSandboxFilter{})
	if err != nil {
		return nil, err
	}
	podSandboxId2Names := make(map[string]string, len(criPodSandboxes))
	for _, podSandbox := range criPodSandboxes {
		if podSandbox.Metadata != nil && podSandbox.Metadata.Name != "" {
			podSandboxId2Names[podSandbox.Id] = podSandbox.Metadata.Name
		}
	}

	criContainers, err := service.ListContainers(containerFilter)
	if err != nil {
		return nil, err
	}

	var containers []model.Container
	for _, criContainer := range criContainers {
		container := model.Container{
			Id: criContainer.Id,
			PodId: criContainer.PodSandboxId,
			RuntimeName: runtimeName,
			State: containerState2String(criContainer.State),
			DataSource: model.ViaCRI,
		}
		if criContainer.Metadata != nil {
			container.Name = criContainer.Metadata.Name
		}
		if podSandboxName, ok := podSandboxId2Names[container.PodId]; ok {
			container.PodName = podSandboxName
		}
		containers = append(containers, container)
	}
	return containers, nil
}

func containerState2String(state runtimeapis.ContainerState) string {
	return strings.TrimPrefix(state.String(), "CONTAINER_")
}
