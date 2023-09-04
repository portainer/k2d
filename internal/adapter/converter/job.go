package converter

import (
	"github.com/docker/docker/api/types"
	batchv1 "k8s.io/api/batch/v1"
)

// ConvertContainerToPod converts a given Docker container into a Kubernetes Pod object.
// TODO: needs full description
func (converter *DockerAPIConverter) ConvertContainerToJob(container types.Container) batchv1.Job {
	panic("unimplemented")
}
