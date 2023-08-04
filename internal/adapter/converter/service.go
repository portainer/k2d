package converter

import (
	"fmt"
	"strconv"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/go-connections/nat"
	k2dtypes "github.com/portainer/k2d/internal/adapter/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

	service.Spec.ExternalIPs = []string{
		converter.k2dServerConfiguration.ServerIpAddr,
	}

	service.Spec.ClusterIPs = []string{container.NetworkSettings.Networks[k2dtypes.K2DNetworkName].IPAddress}
}
