package converter

import (
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	k2dtypes "github.com/portainer/k2d/internal/adapter/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubernetes/pkg/apis/apps"
	"k8s.io/kubernetes/pkg/apis/core"
)

func (converter *DockerAPIConverter) ConvertContainerToDeployment(container types.Container) apps.Deployment {
	containerName := strings.TrimPrefix(container.Names[0], "/")
	containerState := container.State

	deployment := apps.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:              containerName,
			CreationTimestamp: metav1.NewTime(time.Unix(container.Created, 0)),
			Namespace:         "default",
			Annotations: map[string]string{
				"kubectl.kubernetes.io/last-applied-configuration": container.Labels[k2dtypes.WorkloadLastAppliedConfigLabelKey],
			},
		},
		Spec: apps.DeploymentSpec{
			Replicas: 1,
			Template: core.PodTemplateSpec{
				Spec: core.PodSpec{
					Containers: []core.Container{
						{
							Name:  containerName,
							Image: container.Image,
						},
					},
				},
			},
		},
		Status: apps.DeploymentStatus{
			Replicas: 1,
		},
	}

	if containerState == "running" {
		deployment.Status.UpdatedReplicas = 1
		deployment.Status.ReadyReplicas = 1
		deployment.Status.AvailableReplicas = 1
	} else {
		deployment.Status.UnavailableReplicas = 1
	}

	return deployment
}

func (converter *DockerAPIConverter) ConvertContainersToDeployments(containers []types.Container) apps.DeploymentList {
	deployments := []apps.Deployment{}

	for _, container := range containers {
		deployment := converter.ConvertContainerToDeployment(container)
		deployments = append(deployments, deployment)
	}

	return apps.DeploymentList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "DeploymentList",
			APIVersion: "apps/v1",
		},
		Items: deployments,
	}
}
