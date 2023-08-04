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
	// StatefulSetWorkloadType is the label value used to identify a StatefulSet workload
	// It is stored on a container as a label and used to filter containers when listing statefulsets
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

func (adapter *KubeDockerAdapter) ListStatefulSets(ctx context.Context) (appsv1.StatefulSetList, error) {
	statefulSetList, err := adapter.listStatefulSets(ctx)
	if err != nil {
		return appsv1.StatefulSetList{}, fmt.Errorf("unable to list statefulsets: %w", err)
	}

	versionedStatefulSetList := appsv1.StatefulSetList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "StatefulSetList",
			APIVersion: "apps/v1",
		},
	}

	err = adapter.ConvertK8SResource(&statefulSetList, &versionedStatefulSetList)
	if err != nil {
		return appsv1.StatefulSetList{}, fmt.Errorf("unable to convert internal StatefulSetList to versioned StatefulSetList: %w", err)
	}

	return versionedStatefulSetList, nil
}

func (adapter *KubeDockerAdapter) GetStatefulSetTable(ctx context.Context) (*metav1.Table, error) {
	statefulSetList, err := adapter.listStatefulSets(ctx)
	if err != nil {
		return &metav1.Table{}, fmt.Errorf("unable to list statefulsets: %w", err)
	}

	return k8s.GenerateTable(&statefulSetList)
}

func (adapter *KubeDockerAdapter) listStatefulSets(ctx context.Context) (apps.StatefulSetList, error) {
	labelFilter := filters.NewArgs()
	labelFilter.Add("label", fmt.Sprintf("%s=%s", k2dtypes.WorkloadLabelKey, StatefulSetWorkloadType))

	containers, err := adapter.cli.ContainerList(ctx, types.ContainerListOptions{All: true, Filters: labelFilter})
	if err != nil {
		return apps.StatefulSetList{}, fmt.Errorf("unable to list containers: %w", err)
	}

	return adapter.converter.ConvertContainersToStatefulSets(containers), nil
}
