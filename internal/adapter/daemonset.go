package adapter

import (
	"context"
	"fmt"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"k8s.io/kubernetes/pkg/apis/apps"

	k2dtypes "github.com/portainer/k2d/internal/adapter/types"
	appsv1 "k8s.io/api/apps/v1"
)

const (
	// DaemonSetWorkloadType is the label value used to identify a DaemonSet workload
	DaemonSetWorkloadType = "daemonset"
)

func (adapter *KubeDockerAdapter) CreateContainerFromDaemonSet(ctx context.Context, daemonSet *appsv1.DaemonSet) error {
	opts := ContainerCreationOptions{
		containerName: daemonSet.Name,
		podSpec:       daemonSet.Spec.Template.Spec,
		labels:        daemonSet.Spec.Template.Labels,
	}

	opts.labels[k2dtypes.WorkloadLabelKey] = DaemonSetWorkloadType

	return adapter.createContainerFromPodSpec(ctx, opts)
}

func (adapter *KubeDockerAdapter) ListDaemonSets(ctx context.Context) (apps.DaemonSetList, error) {
	labelFilter := filters.NewArgs()
	labelFilter.Add("label", fmt.Sprintf("%s=%s", k2dtypes.WorkloadLabelKey, DaemonSetWorkloadType))

	containers, err := adapter.cli.ContainerList(ctx, types.ContainerListOptions{All: true, Filters: labelFilter})
	if err != nil {
		return apps.DaemonSetList{}, fmt.Errorf("unable to list containers: %w", err)
	}

	daemonSetList := adapter.converter.ConvertContainersToDaemonSets(containers)

	return daemonSetList, nil
}
