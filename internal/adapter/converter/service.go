package converter

import (
	"fmt"
	"time"

	"github.com/docker/docker/api/types"
	k2dtypes "github.com/portainer/k2d/internal/adapter/types"
	"github.com/portainer/k2d/pkg/network"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/kubernetes/pkg/apis/core"
)

func (converter *DockerAPIConverter) ConvertContainerToService(container types.Container) core.Service {
	service := core.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:              container.Labels[k2dtypes.ServiceNameLabelKey],
			CreationTimestamp: metav1.NewTime(time.Unix(container.Created, 0)),
			Namespace:         "default",
			Annotations: map[string]string{
				"kubectl.kubernetes.io/last-applied-configuration": container.Labels[k2dtypes.ServiceLastAppliedConfigLabelKey],
			},
		},
		Spec: core.ServiceSpec{
			Type:  core.ServiceTypeClusterIP,
			Ports: []core.ServicePort{},
			ExternalIPs: []string{
				converter.k2dServerConfiguration.ServerIpAddr,
			},
		},
	}

	_, ok := container.NetworkSettings.Networks[k2dtypes.K2DNetworkName]
	if ok {
		service.Spec.ClusterIPs = []string{container.NetworkSettings.Networks[k2dtypes.K2DNetworkName].IPAddress}
	}

	if len(container.Ports) > 0 {
		service.Spec.Type = core.ServiceTypeNodePort

		for _, port := range container.Ports {
			// Skip ip v6 ports
			if network.IsIpV6(port.IP) {
				continue
			}

			service.Spec.Ports = append(service.Spec.Ports, core.ServicePort{
				Name:       fmt.Sprintf("%d-%s", port.PrivatePort, port.Type),
				Protocol:   core.Protocol(port.Type),
				Port:       int32(port.PrivatePort),
				TargetPort: intstr.FromInt(int(port.PrivatePort)),
			})
		}
	}

	return service
}

func (converter *DockerAPIConverter) ConvertContainersToServices(containers []types.Container) core.ServiceList {
	services := []core.Service{}

	for _, container := range containers {
		if container.Labels[k2dtypes.ServiceNameLabelKey] == "" {
			continue
		}

		service := converter.ConvertContainerToService(container)
		services = append(services, service)
	}

	return core.ServiceList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ServiceList",
			APIVersion: "v1",
		},
		Items: services,
	}
}
