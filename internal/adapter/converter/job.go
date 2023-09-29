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

	job.ObjectMeta.Annotations["kubectl.kubernetes.io/last-applied-configuration"] = container.Labels[k2dtypes.LastAppliedConfigLabelKey]

	containerState := container.State

	startTime, _ := time.Parse(time.RFC3339Nano, json.State.StartedAt)

	metaStartTime := &metav1.Time{
		Time: startTime,
	}

	job.Status.StartTime = metaStartTime

	job.Status.Active = 0

	if containerState == "running" {
		job.Status.Active = 1
	} else {
		if json.State.ExitCode == 0 {
			job.Status.Succeeded = 1

			completionTime, _ := time.Parse(time.RFC3339Nano, json.State.FinishedAt)

			metaCompletionTime := &metav1.Time{
				Time: completionTime,
			}

			job.Status.CompletionTime = metaCompletionTime
		} else {
			job.Status.Failed = 1
		}
	}
}
