package adapter

import (
	"context"
	"fmt"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubernetes/pkg/apis/apps"

	k2dtypes "github.com/portainer/k2d/internal/adapter/types"
	appsv1 "k8s.io/api/apps/v1"
)

const (
	// DeploymentWorkloadType is the label value used to identify a Deployment workload
	DeploymentWorkloadType = "deployment"
)

func (adapter *KubeDockerAdapter) CreateContainerFromDeployment(ctx context.Context, deployment *appsv1.Deployment) error {
	opts := ContainerCreationOptions{
		containerName: deployment.Name,
		podSpec:       deployment.Spec.Template.Spec,
		labels:        deployment.Spec.Template.Labels,
	}

	opts.labels[k2dtypes.WorkloadLabelKey] = DeploymentWorkloadType
	opts.lastAppliedConfiguration = deployment.ObjectMeta.Annotations["kubectl.kubernetes.io/last-applied-configuration"]

	return adapter.createContainerFromPodSpec(ctx, opts)
}

func (adapter *KubeDockerAdapter) GetDeployment(ctx context.Context, deploymentName string) (*appsv1.Deployment, error) {
	containers, err := adapter.cli.ContainerList(ctx, types.ContainerListOptions{All: true})
	if err != nil {
		return nil, fmt.Errorf("unable to list containers: %w", err)
	}

	for _, container := range containers {
		if container.Names[0] == "/"+deploymentName {
			deployment := adapter.converter.ConvertContainerToDeployment(container)

			versionedDeployment := appsv1.Deployment{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Deployment",
					APIVersion: "apps/v1",
				},
			}

			err := adapter.ConvertObjectToVersionedObject(&deployment, &versionedDeployment)
			if err != nil {
				return nil, fmt.Errorf("unable to convert object to versioned object: %w", err)
			}

			return &versionedDeployment, nil
		}
	}

	return nil, nil
}

func (adapter *KubeDockerAdapter) ListDeployments(ctx context.Context) (apps.DeploymentList, error) {
	labelFilter := filters.NewArgs()
	labelFilter.Add("label", fmt.Sprintf("%s=%s", k2dtypes.WorkloadLabelKey, DeploymentWorkloadType))

	containers, err := adapter.cli.ContainerList(ctx, types.ContainerListOptions{All: true, Filters: labelFilter})
	if err != nil {
		return apps.DeploymentList{}, fmt.Errorf("unable to list containers: %w", err)
	}

	deploymentList := adapter.converter.ConvertContainersToDeployments(containers)

	return deploymentList, nil
}
