package converter

import (
	"time"

	"github.com/docker/docker/api/types"
	k2dtypes "github.com/portainer/k2d/internal/adapter/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubernetes/pkg/apis/batch"
)

func (converter *DockerAPIConverter) UpdateJobFromContainerInfo(job *batch.Job, container types.Container, json types.ContainerJSON) {
	job.TypeMeta = metav1.TypeMeta{
		Kind:       "Job",
		APIVersion: "batch/v1",
	}

	job.ObjectMeta.CreationTimestamp = metav1.NewTime(time.Unix(container.Created, 0))
	if job.ObjectMeta.Annotations == nil {
		job.ObjectMeta.Annotations = make(map[string]string)
	}

	job.ObjectMeta.Annotations["kubectl.kubernetes.io/last-applied-configuration"] = container.Labels[k2dtypes.WorkloadLastAppliedConfigLabelKey]

	containerState := container.State

	job.Status.Active = 0

	if containerState == "running" {
		job.Status.Active = 1
	} else {
		// TODO: handle completion status?
		if json.State.ExitCode == 0 {
			job.Status.Succeeded = 1
		} else {
			job.Status.Failed = 1
		}

	}

	// TODO: handle duration?
	// /containers/<container ID>/json ? This will allow getting:
	// - State.ExitCode
	// - State.StartedAt
	// - State.FinishedAt
	job.Status.CompletionTime.Time, _ = time.Parse("", json.State.FinishedAt)
}
