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
	// StatefulSetWorkloadType is the label value used to identify a StatefulSet workload
	StatefulSetWorkloadType = "statefulset"
)

func (adapter *KubeDockerAdapter) CreateContainerFromStatefulSet(ctx context.Context, statefulset *appsv1.StatefulSet) error {
	opts := ContainerCreationOptions{
		containerName: statefulset.Name,
		podSpec:       statefulset.Spec.Template.Spec,
		labels:        statefulset.Spec.Template.Labels,
	}

	opts.labels[k2dtypes.WorkloadLabelKey] = StatefulSetWorkloadType

	return adapter.createContainerFromPodSpec(ctx, opts)
}

func (adapter *KubeDockerAdapter) ListStatefulSets(ctx context.Context) (apps.StatefulSetList, error) {
	labelFilter := filters.NewArgs()
	labelFilter.Add("label", fmt.Sprintf("%s=%s", k2dtypes.WorkloadLabelKey, StatefulSetWorkloadType))

	containers, err := adapter.cli.ContainerList(ctx, types.ContainerListOptions{All: true, Filters: labelFilter})
	if err != nil {
		return apps.StatefulSetList{}, fmt.Errorf("unable to list containers: %w", err)
	}

	statefulSetList := adapter.converter.ConvertContainersToStatefulSets(containers)

	return statefulSetList, nil
}
