package adapter

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubernetes/pkg/apis/apps"

	adaptererr "github.com/portainer/k2d/internal/adapter/errors"
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
		networkName:   deployment.Namespace,
		podSpec:       deployment.Spec.Template.Spec,
		labels:        deployment.Spec.Template.Labels,
	}

	opts.labels[k2dtypes.WorkloadLabelKey] = DeploymentWorkloadType

	if deployment.Labels["app.kubernetes.io/managed-by"] == "Helm" {
		deploymentData, err := json.Marshal(deployment)
		if err != nil {
			return fmt.Errorf("unable to marshal deployment: %w", err)
		}
		deployment.ObjectMeta.Annotations["kubectl.kubernetes.io/last-applied-configuration"] = string(deploymentData)
	}

	// kubectl create deployment does not pass the last-applied-configuration annotation
	// so we need to add it manually
	if deployment.ObjectMeta.Annotations["kubectl.kubernetes.io/last-applied-configuration"] == "" {
		deploymentData, err := json.Marshal(deployment)
		if err != nil {
			return fmt.Errorf("unable to marshal deployment: %w", err)
		}
		opts.labels[k2dtypes.WorkloadLastAppliedConfigLabelKey] = string(deploymentData)
	}

	opts.lastAppliedConfiguration = deployment.ObjectMeta.Annotations["kubectl.kubernetes.io/last-applied-configuration"]

	return adapter.createContainerFromPodSpec(ctx, opts)
}

func (adapter *KubeDockerAdapter) getContainerFromDeploymentName(ctx context.Context, deploymentName, namespace string) (types.Container, error) {
	labelFilter := filters.NewArgs()
	labelFilter.Add("label", fmt.Sprintf("%s=%s", k2dtypes.WorkloadLabelKey, DeploymentWorkloadType))
	labelFilter.Add("label", fmt.Sprintf("%s=%s", k2dtypes.NamespaceLabelKey, namespace))
	labelFilter.Add("label", fmt.Sprintf("%s=%s", k2dtypes.WorkloadNameLabelKey, deploymentName))

	containers, err := adapter.cli.ContainerList(ctx, types.ContainerListOptions{All: true, Filters: labelFilter})
	if err != nil {
		return types.Container{}, fmt.Errorf("unable to list containers: %w", err)
	}

	if len(containers) == 0 {
		return types.Container{}, adaptererr.ErrResourceNotFound
	}

	if len(containers) > 1 {
		return types.Container{}, fmt.Errorf("multiple containers were found with the associated deployment name %s", deploymentName)
	}

	return containers[0], nil
}

func (adapter *KubeDockerAdapter) GetDeployment(ctx context.Context, deploymentName string, namespace string) (*appsv1.Deployment, error) {
	container, err := adapter.getContainerFromDeploymentName(ctx, deploymentName, namespace)
	if err != nil {
		return nil, fmt.Errorf("unable to get container from deployment name: %w", err)
	}

	deployment, err := adapter.buildDeploymentFromContainer(container)
	if err != nil {
		return nil, fmt.Errorf("unable to get deployment: %w", err)
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

func (adapter *KubeDockerAdapter) GetDeploymentTable(ctx context.Context, namespaceName string) (*metav1.Table, error) {
	deploymentList, err := adapter.listDeployments(ctx, namespaceName)
	if err != nil {
		return &metav1.Table{}, fmt.Errorf("unable to list deployments: %w", err)
	}

	return k8s.GenerateTable(&deploymentList)
}

func (adapter *KubeDockerAdapter) ListDeployments(ctx context.Context, namespaceName string) (appsv1.DeploymentList, error) {
	deploymentList, err := adapter.listDeployments(ctx, namespaceName)
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

func (adapter *KubeDockerAdapter) buildDeploymentFromContainer(container types.Container) (*apps.Deployment, error) {

	if container.Labels[k2dtypes.WorkloadLastAppliedConfigLabelKey] == "" {
		return nil, fmt.Errorf("unable to build deployment, missing %s label on container %s", k2dtypes.WorkloadLastAppliedConfigLabelKey, container.Names[0])
	}

	deploymentData := container.Labels[k2dtypes.WorkloadLastAppliedConfigLabelKey]

	versionedDeployment := appsv1.Deployment{}

	err := json.Unmarshal([]byte(deploymentData), &versionedDeployment)
	if err != nil {
		return nil, fmt.Errorf("unable to unmarshal versioned deployment: %w", err)
	}

	deployment := apps.Deployment{}
	err = adapter.ConvertK8SResource(&versionedDeployment, &deployment)
	if err != nil {
		return nil, fmt.Errorf("unable to convert versioned deployment spec to internal deployment spec: %w", err)
	}

	adapter.converter.UpdateDeploymentFromContainerInfo(&deployment, container)

	return &deployment, nil
}

func (adapter *KubeDockerAdapter) listDeployments(ctx context.Context, namespaceName string) (apps.DeploymentList, error) {
	labelFilter := filters.NewArgs()
	labelFilter.Add("label", fmt.Sprintf("%s=%s", k2dtypes.WorkloadLabelKey, DeploymentWorkloadType))
	labelFilter.Add("label", fmt.Sprintf("%s=%s", k2dtypes.NamespaceLabelKey, namespaceName))

	containers, err := adapter.cli.ContainerList(ctx, types.ContainerListOptions{All: true, Filters: labelFilter})
	if err != nil {
		return apps.DeploymentList{}, fmt.Errorf("unable to list containers: %w", err)
	}

	deployments := []apps.Deployment{}

	for _, container := range containers {
		deployment, err := adapter.buildDeploymentFromContainer(container)
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
