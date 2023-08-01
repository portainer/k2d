package adapter

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/go-connections/nat"
	k2dtypes "github.com/portainer/k2d/internal/adapter/types"
	"github.com/portainer/k2d/internal/logging"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubernetes/pkg/apis/core"
)

func (adapter *KubeDockerAdapter) CreateContainerFromService(ctx context.Context, service *corev1.Service) error {
	// TODO: headless service should be ignored
	// should return immediately

	containers, err := adapter.cli.ContainerList(ctx, types.ContainerListOptions{})
	if err != nil {
		return fmt.Errorf("unable to list containers: %w", err)
	}

	matchingContainer := findMatchingContainer(containers, service.Spec.Selector)

	if matchingContainer == nil {
		return errors.New("no container was found matching the service selector")
	}

	logger := logging.LoggerFromContext(ctx)

	if service.ObjectMeta.Annotations["kubectl.kubernetes.io/last-applied-configuration"] == matchingContainer.Labels[k2dtypes.ServiceLastAppliedConfigLabelKey] {
		logger.Infow("the container matching the service selector already exists with the same service configuration. The update will be skipped",
			"container_id", matchingContainer.ID,
			"service_name", service.Name,
		)
		return nil
	}

	logger.Infow("container found matching the service selector with a different service configuration. The container will be re-created",
		"container_id", matchingContainer.ID,
	)

	containerDetails, err := adapter.cli.ContainerInspect(ctx, matchingContainer.ID)
	if err != nil {
		return fmt.Errorf("unable to inspect container: %w", err)
	}

	containerStopTimeout := 3
	err = adapter.cli.ContainerStop(ctx, matchingContainer.ID, container.StopOptions{Timeout: &containerStopTimeout})
	if err != nil {
		return fmt.Errorf("unable to stop existing container: %w", err)
	}

	// TODO: should have a converter that takes a service spec and an existing container configuration

	containerConfig := &container.Config{
		Image:        containerDetails.Image,
		Labels:       containerDetails.Config.Labels,
		ExposedPorts: nat.PortSet{},
		Env:          containerDetails.Config.Env,
		User:         containerDetails.Config.User,
	}

	containerConfig.Labels[k2dtypes.ServiceNameLabelKey] = service.Name
	containerConfig.Labels[k2dtypes.ServiceLastAppliedConfigLabelKey] = service.ObjectMeta.Annotations["kubectl.kubernetes.io/last-applied-configuration"]

	hostConfig := &container.HostConfig{
		PortBindings:  nat.PortMap{},
		RestartPolicy: containerDetails.HostConfig.RestartPolicy,
		Binds:         containerDetails.HostConfig.Binds,
		ExtraHosts:    containerDetails.HostConfig.ExtraHosts,
		Privileged:    containerDetails.HostConfig.Privileged,
	}

	networkConfig := &network.NetworkingConfig{
		EndpointsConfig: map[string]*network.EndpointSettings{
			k2dtypes.K2DNetworkName: {},
		},
	}

	for _, port := range service.Spec.Ports {
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

		hostConfig.PortBindings[containerPort] = []nat.PortBinding{hostBinding}
		containerConfig.ExposedPorts[containerPort] = struct{}{}
	}

	err = adapter.cli.ContainerRemove(ctx, matchingContainer.ID, types.ContainerRemoveOptions{})
	if err != nil {
		return err
	}

	// TODO: should only remove previous container if no error during create/start
	// otherwise rollback to old container
	containerCreateResponse, err := adapter.cli.ContainerCreate(ctx, containerConfig, hostConfig, networkConfig, nil, matchingContainer.Names[0])
	if err != nil {
		return fmt.Errorf("unable to create container: %w", err)
	}

	return adapter.cli.ContainerStart(ctx, containerCreateResponse.ID, types.ContainerStartOptions{})
}

func (adapter *KubeDockerAdapter) GetService(ctx context.Context, serviceName string) (*corev1.Service, error) {
	containers, err := adapter.cli.ContainerList(ctx, types.ContainerListOptions{All: true})
	if err != nil {
		return nil, fmt.Errorf("unable to list containers: %w", err)
	}

	for _, container := range containers {
		if container.Labels[k2dtypes.ServiceNameLabelKey] == serviceName {
			pod := adapter.converter.ConvertContainerToService(container)

			versionedService := corev1.Service{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Service",
					APIVersion: "v1",
				},
			}

			err := adapter.ConvertObjectToVersionedObject(&pod, &versionedService)
			if err != nil {
				return nil, fmt.Errorf("unable to convert object to versioned object: %w", err)
			}

			return &versionedService, nil
		}
	}

	return nil, nil
}

func (adapter *KubeDockerAdapter) ListServices(ctx context.Context) (core.ServiceList, error) {
	containers, err := adapter.cli.ContainerList(ctx, types.ContainerListOptions{All: true})
	if err != nil {
		return core.ServiceList{}, fmt.Errorf("unable to list containers: %w", err)
	}

	serviceList := adapter.converter.ConvertContainersToServices(containers)

	return serviceList, nil
}
