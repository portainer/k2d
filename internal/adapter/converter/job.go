package converter

import (
	"github.com/docker/docker/api/types"
	"k8s.io/kubernetes/pkg/apis/batch"
)

// ConvertContainerToPod converts a given Docker container into a Kubernetes Pod object.
// TODO: needs full description
func (converter *DockerAPIConverter) ConvertContainerToJob(container types.Container) batch.Job {
	panic("unimplemented")
}
