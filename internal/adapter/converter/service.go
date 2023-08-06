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
	// if service type is not specified from the YAML file, we default to ClusterIP
	if serviceSpec.Type == "" {
		serviceSpec.Type = core.ServiceTypeClusterIP
	}

	// portBindings forces a random high port to be used for a non-NodePort service
	// hence, we need to check for the NodePort service type only
	for _, port := range serviceSpec.Ports {
		if serviceSpec.Type == core.ServiceTypeNodePort {
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
				// TODO: this requires a check to ensure the randomly selected port is not already in use
				hostBinding.HostPort = strconv.Itoa(rand.Intn(32767-30000+1) + 30000)
			}

			containerCfg.HostConfig.PortBindings[containerPort] = []nat.PortBinding{hostBinding}
			containerCfg.ContainerConfig.ExposedPorts[containerPort] = struct{}{}
		}
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

	if service.Spec.Type == core.ServiceTypeNodePort {
		service.Spec.ExternalIPs = []string{converter.k2dServerConfiguration.ServerIpAddr}

		servicePorts := []core.ServicePort{}
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
	}
}
