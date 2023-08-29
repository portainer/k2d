package adapter

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	k2dtypes "github.com/portainer/k2d/internal/adapter/types"
	"github.com/portainer/k2d/internal/k8s"
	"github.com/portainer/k2d/internal/logging"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubernetes/pkg/apis/core"
)

func (adapter *KubeDockerAdapter) DeleteService(ctx context.Context, serviceName string) error {
	containers, err := adapter.cli.ContainerList(ctx, types.ContainerListOptions{All: true})
	if err != nil {
		return fmt.Errorf("unable to list containers: %w", err)
	}

	logger := logging.LoggerFromContext(ctx)

	for _, cntr := range containers {
		if cntr.Labels[k2dtypes.ServiceNameLabelKey] == serviceName {

			logger.Infow("found the container with the associated service. The container will be re-created and the associated service configuration will be removed.",
				"container_id", cntr.ID,
				"service_name", serviceName,
			)

			cfg, err := adapter.buildContainerConfigurationFromExistingContainer(ctx, cntr.ID)
			if err != nil {
				return fmt.Errorf("unable to build container configuration from existing container: %w", err)
			}

			delete(cfg.ContainerConfig.Labels, k2dtypes.ServiceNameLabelKey)
			delete(cfg.ContainerConfig.Labels, k2dtypes.ServiceLastAppliedConfigLabelKey)

			networkName := cfg.ContainerConfig.Labels[k2dtypes.NamespaceLabelKey]
			if networkName == "default" {
				networkName = "k2d_net"
			}

			cfg.NetworkConfig.EndpointsConfig[networkName].Aliases = []string{}

			return adapter.reCreateContainerWithNewConfiguration(ctx, cntr.ID, cfg)
		}
	}

	logger.Infow("no container was found with the associated service.",
		"service_name", serviceName,
	)

	return nil
}

func (adapter *KubeDockerAdapter) CreateContainerFromService(ctx context.Context, service *corev1.Service) error {
	logger := logging.LoggerFromContext(ctx)

	// headless services are not supported
	if service.Spec.ClusterIP == "None" {
		logger.Infow("headless service detected. The service will be ignored",
			"service_name", service.Name,
		)
		return nil
	}

	// ExternalName services are not supported
	if service.Spec.Type == corev1.ServiceTypeExternalName {
		logger.Infow("externalName service detected. The service will be ignored",
			"service_name", service.Name,
		)
		return nil
	}

	containers, err := adapter.cli.ContainerList(ctx, types.ContainerListOptions{})
	if err != nil {
		return fmt.Errorf("unable to list containers: %w", err)
	}

	matchingContainer := findContainerMatchingSelector(containers, service.Spec.Selector)

	if matchingContainer == nil {
		return errors.New("no container was found matching the service selector")
	}

	if service.Labels["app.kubernetes.io/managed-by"] == "Helm" {
		serviceData, err := json.Marshal(service)
		if err != nil {
			return fmt.Errorf("unable to marshal service: %w", err)
		}
		service.ObjectMeta.Annotations["kubectl.kubernetes.io/last-applied-configuration"] = string(serviceData)
	}

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

	cfg, err := adapter.buildContainerConfigurationFromExistingContainer(ctx, matchingContainer.ID)
	if err != nil {
		return fmt.Errorf("unable to build container configuration from existing container: %w", err)
	}

	cfg.ContainerConfig.Labels[k2dtypes.ServiceNameLabelKey] = service.Name
	cfg.ContainerConfig.Labels[k2dtypes.ServiceLastAppliedConfigLabelKey] = service.ObjectMeta.Annotations["kubectl.kubernetes.io/last-applied-configuration"]

	internalServiceSpec := core.ServiceSpec{}
	err = adapter.ConvertK8SResource(&service.Spec, &internalServiceSpec)
	if err != nil {
		return fmt.Errorf("unable to convert versioned service spec to internal service spec: %w", err)
	}

	usedPorts := make(map[int]struct{})
	for _, container := range containers {
		for _, port := range container.Ports {
			usedPorts[int(port.PublicPort)] = struct{}{}
		}
	}

	err = adapter.converter.ConvertServiceSpecIntoContainerConfiguration(internalServiceSpec, &cfg, usedPorts)
	if err != nil {
		return fmt.Errorf("unable to convert service spec into container configuration: %w", err)
	}

	network := service.Namespace
	if network == "default" {
		network = "k2d_net"
	}

	cfg.NetworkConfig.EndpointsConfig[network].Aliases = []string{
		service.Name,
		fmt.Sprintf("%s.%s", service.Name, service.Namespace),
		fmt.Sprintf("%s.%s.svc", service.Name, service.Namespace),
		fmt.Sprintf("%s.%s.svc.cluster.local", service.Name, service.Namespace),
	}

	return adapter.reCreateContainerWithNewConfiguration(ctx, matchingContainer.ID, cfg)
}

func (adapter *KubeDockerAdapter) GetService(ctx context.Context, serviceName string) (*corev1.Service, error) {
	containers, err := adapter.cli.ContainerList(ctx, types.ContainerListOptions{All: true})
	if err != nil {
		return nil, fmt.Errorf("unable to list containers: %w", err)
	}

	for _, container := range containers {
		if container.Labels[k2dtypes.ServiceNameLabelKey] == serviceName {
			// run an inspect to fetch the error state
			containerInspect, err := adapter.cli.ContainerInspect(ctx, container.ID)
			if err != nil {
				return nil, fmt.Errorf("unable to inspect container for filling in the service status: %w", err)
			}

			if containerInspect.State.Error != "" {
				container.Labels[k2dtypes.ServiceStatusErrorMessage] = containerInspect.State.Error
			}

			service, err := adapter.getService(container)
			if err != nil {
				return nil, fmt.Errorf("unable to get service: %w", err)
			}

			if service == nil {
				return nil, nil
			}

			versionedService := corev1.Service{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Service",
					APIVersion: "v1",
				},
			}

			err = adapter.ConvertK8SResource(service, &versionedService)
			if err != nil {
				return nil, fmt.Errorf("unable to convert internal object to versioned object: %w", err)
			}

			return &versionedService, nil
		}
	}

	return nil, nil
}

func (adapter *KubeDockerAdapter) GetServiceTable(ctx context.Context, namespaceName string) (*metav1.Table, error) {
	serviceList, err := adapter.listServices(ctx, namespaceName)
	if err != nil {
		return &metav1.Table{}, fmt.Errorf("unable to list services: %w", err)
	}

	return k8s.GenerateTable(&serviceList)
}

func (adapter *KubeDockerAdapter) ListServices(ctx context.Context, namespaceName string) (corev1.ServiceList, error) {
	serviceList, err := adapter.listServices(ctx, namespaceName)
	if err != nil {
		return corev1.ServiceList{}, fmt.Errorf("unable to list services: %w", err)
	}

	versionedServiceList := corev1.ServiceList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ServiceList",
			APIVersion: "v1",
		},
	}

	err = adapter.ConvertK8SResource(&serviceList, &versionedServiceList)
	if err != nil {
		return corev1.ServiceList{}, fmt.Errorf("unable to convert internal ServiceList to versioned ServiceList: %w", err)
	}

	return versionedServiceList, nil
}

func (adapter *KubeDockerAdapter) getService(container types.Container) (*core.Service, error) {
	service := core.Service{}

	if container.Labels[k2dtypes.ServiceLastAppliedConfigLabelKey] != "" {
		serviceData := container.Labels[k2dtypes.ServiceLastAppliedConfigLabelKey]

		versionedService := corev1.Service{}

		err := json.Unmarshal([]byte(serviceData), &versionedService)
		if err != nil {
			return nil, fmt.Errorf("unable to unmarshal versioned service: %w", err)
		}

		err = adapter.ConvertK8SResource(&versionedService, &service)
		if err != nil {
			return nil, fmt.Errorf("unable to convert versioned service spec to internal service spec: %w", err)
		}

		adapter.converter.UpdateServiceFromContainerInfo(&service, container)
	} else {
		adapter.logger.Errorf("unable to build service, missing %s label on container %s", k2dtypes.ServiceLastAppliedConfigLabelKey, container.Names[0])
		return nil, nil
	}

	return &service, nil
}

func (adapter *KubeDockerAdapter) listServices(ctx context.Context, namespaceName string) (core.ServiceList, error) {
	labelFilter := filters.NewArgs()
	labelFilter.Add("label", k2dtypes.ServiceNameLabelKey)
	labelFilter.Add("label", fmt.Sprintf("%s=%s", k2dtypes.NamespaceLabelKey, namespaceName))

	containers, err := adapter.cli.ContainerList(ctx, types.ContainerListOptions{All: true, Filters: labelFilter})
	if err != nil {
		return core.ServiceList{}, fmt.Errorf("unable to list containers: %w", err)
	}

	services := []core.Service{}

	for _, container := range containers {
		service, err := adapter.getService(container)
		if err != nil {
			return core.ServiceList{}, fmt.Errorf("unable to get service: %w", err)
		}

		if service != nil {
			services = append(services, *service)
		}
	}

	serviceList := core.ServiceList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ServiceList",
			APIVersion: "v1",
		},
		Items: services,
	}

	return serviceList, nil
}
