package converter

import (
	"fmt"
	"strconv"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/go-connections/nat"
	k2dtypes "github.com/portainer/k2d/internal/adapter/types"
	"github.com/portainer/k2d/pkg/network"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/kubernetes/pkg/apis/core"
)

func (converter *DockerAPIConverter) ConvertServiceSpecIntoContainerConfiguration(serviceSpec corev1.ServiceSpec, containerCfg *ContainerConfiguration) error {
	for _, port := range serviceSpec.Ports {
		containerPort, err := nat.NewPort(string(port.Protocol), port.TargetPort.String())
		if err != nil {
			return fmt.Errorf("invalid container port: %w", err)
		}

		hostBinding := nat.PortBinding{
			HostIP: "0.0.0.0",
		}

		if port.NodePort != 0 {
			hostBinding.HostPort = strconv.Itoa(int(port.NodePort))
		} else {
			hostBinding.HostPort = strconv.Itoa(int(port.Port))
		}

		containerCfg.HostConfig.PortBindings[containerPort] = []nat.PortBinding{hostBinding}
		containerCfg.ContainerConfig.ExposedPorts[containerPort] = struct{}{}
	}

	return nil
}

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
