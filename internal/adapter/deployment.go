package adapter

import (
	"context"
	"encoding/json"
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
	// DeploymentWorkloadType is the label value used to identify a Deployment workload
	// It is stored on a container as a label and used to filter containers when listing deployments
	DeploymentWorkloadType = "deployment"
)

func (adapter *KubeDockerAdapter) CreateContainerFromDeployment(ctx context.Context, deployment *appsv1.Deployment) error {
	opts := ContainerCreationOptions{
		containerName: deployment.Name,
		podSpec:       deployment.Spec.Template.Spec,
		labels:        deployment.Spec.Template.Labels,
	}

	if deployment.ObjectMeta.Annotations["kubectl.kubernetes.io/last-applied-configuration"] == "" && deployment.Labels["app.kubernetes.io/managed-by"] == "Helm" {
		deploymentData, err := json.Marshal(deployment)
		if err != nil {
			return fmt.Errorf("unable to marshal deployment: %w", err)
		}
		deployment.ObjectMeta.Annotations["kubectl.kubernetes.io/last-applied-configuration"] = string(deploymentData)
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
			deployment, err := adapter.getDeployment(container)
			if err != nil {
				return nil, fmt.Errorf("unable to get deployment: %w", err)
			}

			if deployment == nil {
				return nil, nil
			}

			versionedDeployment := appsv1.Deployment{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Deployment",
					APIVersion: "apps/v1",
				},
			}

			err = adapter.ConvertK8SResource(deployment, &versionedDeployment)
			if err != nil {
				return nil, fmt.Errorf("unable to convert internal object to versioned object: %w", err)
			}

			return &versionedDeployment, nil
		}
	}

	return nil, nil
}

func (adapter *KubeDockerAdapter) GetDeploymentTable(ctx context.Context) (*metav1.Table, error) {
	deploymentList, err := adapter.listDeployments(ctx)
	if err != nil {
		return &metav1.Table{}, fmt.Errorf("unable to list deployments: %w", err)
	}

	return k8s.GenerateTable(&deploymentList)
}

func (adapter *KubeDockerAdapter) ListDeployments(ctx context.Context) (appsv1.DeploymentList, error) {
	deploymentList, err := adapter.listDeployments(ctx)
	if err != nil {
		return appsv1.DeploymentList{}, fmt.Errorf("unable to list deployments: %w", err)
	}

	versionedDeploymentList := appsv1.DeploymentList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "DeploymentList",
			APIVersion: "apps/v1",
		},
	}

	err = adapter.ConvertK8SResource(&deploymentList, &versionedDeploymentList)
	if err != nil {
		return appsv1.DeploymentList{}, fmt.Errorf("unable to convert internal DeploymentList to versioned DeploymentList: %w", err)
	}

	return versionedDeploymentList, nil
}

func (adapter *KubeDockerAdapter) getDeployment(container types.Container) (*apps.Deployment, error) {
	deployment := apps.Deployment{}

	if container.Labels[k2dtypes.WorkloadLastAppliedConfigLabelKey] != "" {
		deploymentData := container.Labels[k2dtypes.WorkloadLastAppliedConfigLabelKey]

		versionedDeployment := appsv1.Deployment{}

		err := json.Unmarshal([]byte(deploymentData), &versionedDeployment)
		if err != nil {
			return nil, fmt.Errorf("unable to unmarshal versioned deployment: %w", err)
		}

		err = adapter.ConvertK8SResource(&versionedDeployment, &deployment)
		if err != nil {
			return nil, fmt.Errorf("unable to convert versioned deployment spec to internal deployment spec: %w", err)
		}

		adapter.converter.UpdateDeploymentFromContainerInfo(&deployment, container)

	} else {
		adapter.logger.Errorf("unable to build deployment, missing %s label on container %s", k2dtypes.WorkloadLastAppliedConfigLabelKey, container.Names[0])
		return nil, nil
	}

	return &deployment, nil
}

func (adapter *KubeDockerAdapter) listDeployments(ctx context.Context) (apps.DeploymentList, error) {
	labelFilter := filters.NewArgs()
	labelFilter.Add("label", fmt.Sprintf("%s=%s", k2dtypes.WorkloadLabelKey, DeploymentWorkloadType))

	containers, err := adapter.cli.ContainerList(ctx, types.ContainerListOptions{All: true, Filters: labelFilter})
	if err != nil {
		return apps.DeploymentList{}, fmt.Errorf("unable to list containers: %w", err)
	}

	deployments := []apps.Deployment{}

	for _, container := range containers {
		deployment, err := adapter.getDeployment(container)
		if err != nil {
			return apps.DeploymentList{}, fmt.Errorf("unable to get deployment: %w", err)
		}

		if deployment != nil {
			deployments = append(deployments, *deployment)
		}
	}

	return apps.DeploymentList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "DeploymentList",
			APIVersion: "apps/v1",
		},
		Items: deployments,
	}, nil
}
