package converter

import (
	"time"

	"github.com/docker/docker/api/types"
	k2dtypes "github.com/portainer/k2d/internal/adapter/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubernetes/pkg/apis/core"
)

func (converter *DockerAPIConverter) ConvertNetworkToNamespace(network *types.NetworkResource) *core.Namespace {
	if network.Name == "k2d_net" {
		network.Name = "default"
	}

	namespace := &core.Namespace{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Namespace",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:              network.Name,
			CreationTimestamp: metav1.NewTime(time.Unix(network.Created.Unix(), 0)),
			Annotations: map[string]string{
				"kubectl.kubernetes.io/last-applied-configuration": network.Labels[k2dtypes.NamespaceLastAppliedConfigLabelKey],
			},
		},
		Status: core.NamespaceStatus{
			Phase: core.NamespaceActive,
		},
	}

	return namespace
}
