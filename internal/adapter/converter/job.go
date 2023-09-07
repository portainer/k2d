package converter

import (
	"time"

	"github.com/docker/docker/api/types"
	k2dtypes "github.com/portainer/k2d/internal/adapter/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubernetes/pkg/apis/batch"
)

func (converter *DockerAPIConverter) UpdateJobFromContainerInfo(job *batch.Job, container types.Container) {
	job.TypeMeta = metav1.TypeMeta{
		Kind:       "Job",
		APIVersion: "batch/v1",
	}

	job.ObjectMeta.CreationTimestamp = metav1.NewTime(time.Unix(container.Created, 0))
	if job.ObjectMeta.Annotations == nil {
		job.ObjectMeta.Annotations = make(map[string]string)
	}

	job.ObjectMeta.Annotations["kubectl.kubernetes.io/last-applied-configuration"] = container.Labels[k2dtypes.WorkloadLastAppliedConfigLabelKey]

	// containerState := container.State

	// // if the number of replicas isn't set in the job, set it to 1
	// if job.Spec.Replicas == 0 {
	// 	job.Spec.Replicas = 1
	// }

	// job.Status.Replicas = 1

	// if containerState == "running" {
	// 	job.Status.UpdatedReplicas = 1
	// 	job.Status.ReadyReplicas = 1
	// 	job.Status.AvailableReplicas = 1
	// } else {
	// 	job.Status.UnavailableReplicas = 1
	// }
}
