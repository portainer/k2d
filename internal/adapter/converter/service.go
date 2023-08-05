package converter

import (
	"fmt"
	"math/rand"
	"strconv"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/go-connections/nat"
	k2dtypes "github.com/portainer/k2d/internal/adapter/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	intstr "k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/kubernetes/pkg/apis/core"
)

func (converter *DockerAPIConverter) ConvertServiceSpecIntoContainerConfiguration(serviceSpec core.ServiceSpec, containerCfg *ContainerConfiguration) error {
	for _, port := range serviceSpec.Ports {
		containerPort, err := nat.NewPort(string(port.Protocol), port.TargetPort.String())
		if err != nil {
			return fmt.Errorf("invalid container port: %w", err)
		}

		hostBinding := nat.PortBinding{
			HostIP: "0.0.0.0",
		}

		switch serviceSpec.Type {
		case core.ServiceTypeNodePort:
			// nodePort: this is a port in the range of 30000-32767 that will be open in each node
			// TODO: this requires a check to ensure the port is not already in use
			hostBinding.HostPort = strconv.Itoa(rand.Intn(32767-30000+1) + 30000)
			if port.NodePort != 0 {
				hostBinding.HostPort = strconv.Itoa(int(port.NodePort))
			}
		case core.ServiceTypeClusterIP:
			hostBinding.HostPort = ""
		default:
			hostBinding.HostPort = ""
		}

		containerCfg.HostConfig.PortBindings[containerPort] = []nat.PortBinding{hostBinding}
		containerCfg.ContainerConfig.ExposedPorts[containerPort] = struct{}{}
	}

	return nil
}

func (converter *DockerAPIConverter) UpdateServiceFromContainerInfo(service *core.Service, container types.Container) {
	service.TypeMeta = metav1.TypeMeta{
		Kind:       "Service",
		APIVersion: "v1",
	}

	service.ObjectMeta.CreationTimestamp = metav1.NewTime(time.Unix(container.Created, 0))
	service.ObjectMeta.Annotations = map[string]string{
		"kubectl.kubernetes.io/last-applied-configuration": container.Labels[k2dtypes.ServiceLastAppliedConfigLabelKey],
	}

	if service.Spec.Type == "" {
		service.Spec.Type = core.ServiceTypeClusterIP
	}

	service.Spec.ClusterIPs = []string{container.NetworkSettings.Networks[k2dtypes.K2DNetworkName].IPAddress}

	servicePorts := []core.ServicePort{}
	switch service.Spec.Type {
	case core.ServiceTypeNodePort:
		service.Spec.ExternalIPs = []string{
			converter.k2dServerConfiguration.ServerIpAddr,
		}
		for _, port := range service.Spec.Ports {
			for _, containerPort := range container.Ports {
				if port.TargetPort == intstr.Parse(strconv.Itoa(int(containerPort.PrivatePort))) {
					servicePorts = append(servicePorts, core.ServicePort{
						Name:       port.Name,
						Protocol:   port.Protocol,
						Port:       port.Port,
						TargetPort: port.TargetPort,
						NodePort:   int32(containerPort.PublicPort),
					})
				}
			}
		}
		service.Spec.Ports = servicePorts
	case core.ServiceTypeClusterIP:
		service.Spec.ExternalIPs = []string{}
	default:
		service.Spec.ExternalIPs = []string{}
	}
}
