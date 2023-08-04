package adapter

import (
	"context"
	"fmt"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubernetes/pkg/apis/apps"

	k2dtypes "github.com/portainer/k2d/internal/adapter/types"
	"github.com/portainer/k2d/internal/k8s"
	appsv1 "k8s.io/api/apps/v1"
)

const (
	// DaemonSetWorkloadType is the label value used to identify a DaemonSet workload
	// It is stored on a container as a label and used to filter containers when listing daemonsets
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

func (adapter *KubeDockerAdapter) ListDaemonSets(ctx context.Context) (appsv1.DaemonSetList, error) {
	daemonSetList, err := adapter.listDaemonSets(ctx)
	if err != nil {
		return appsv1.DaemonSetList{}, fmt.Errorf("unable to list daemonsets: %w", err)
	}

	versionedDaemonSetList := appsv1.DaemonSetList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "DaemonSetList",
			APIVersion: "apps/v1",
		},
	}

	err = adapter.ConvertK8SResource(&daemonSetList, &versionedDaemonSetList)
	if err != nil {
		return appsv1.DaemonSetList{}, fmt.Errorf("unable to convert internal DaemonSetList to versioned DaemonSetList: %w", err)
	}

	return versionedDaemonSetList, nil
}

func (adapter *KubeDockerAdapter) GetDaemonSetTable(ctx context.Context) (*metav1.Table, error) {
	daemonSetList, err := adapter.listDaemonSets(ctx)
	if err != nil {
		return &metav1.Table{}, fmt.Errorf("unable to list daemonsets: %w", err)
	}

	return k8s.GenerateTable(&daemonSetList)
}

func (adapter *KubeDockerAdapter) listDaemonSets(ctx context.Context) (apps.DaemonSetList, error) {
	labelFilter := filters.NewArgs()
	labelFilter.Add("label", fmt.Sprintf("%s=%s", k2dtypes.WorkloadLabelKey, DaemonSetWorkloadType))

	containers, err := adapter.cli.ContainerList(ctx, types.ContainerListOptions{All: true, Filters: labelFilter})
	if err != nil {
		return apps.DaemonSetList{}, fmt.Errorf("unable to list containers: %w", err)
	}

	return adapter.converter.ConvertContainersToDaemonSets(containers), nil
}
