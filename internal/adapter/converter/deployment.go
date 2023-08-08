package converter

import (
	"time"

	"github.com/docker/docker/api/types"
	k2dtypes "github.com/portainer/k2d/internal/adapter/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubernetes/pkg/apis/apps"
)

func (converter *DockerAPIConverter) UpdateDeploymentFromContainerInfo(deployment *apps.Deployment, container types.Container) {
	deployment.TypeMeta = metav1.TypeMeta{
		Kind:       "Deployment",
		APIVersion: "apps/v1",
	}

	deployment.ObjectMeta.CreationTimestamp = metav1.NewTime(time.Unix(container.Created, 0))
	deployment.ObjectMeta.Annotations["kubectl.kubernetes.io/last-applied-configuration"] = container.Labels[k2dtypes.WorkloadLastAppliedConfigLabelKey]

	containerState := container.State

	// if the number of replicas isn't set in the deployment, set it to 1
	if deployment.Spec.Replicas == 0 {
		deployment.Spec.Replicas = 1
	}

	deployment.Status.Replicas = 1

	if containerState == "running" {
		deployment.Status.UpdatedReplicas = 1
		deployment.Status.ReadyReplicas = 1
		deployment.Status.AvailableReplicas = 1
	} else {
		deployment.Status.UnavailableReplicas = 1
	}
}
