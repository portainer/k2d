package adapter

import (
	"context"
	"errors"
	"fmt"

	"github.com/docker/docker/api/types"
	k2dtypes "github.com/portainer/k2d/internal/adapter/types"
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

	containers, err := adapter.cli.ContainerList(ctx, types.ContainerListOptions{})
	if err != nil {
		return fmt.Errorf("unable to list containers: %w", err)
	}

	matchingContainer := findContainerMatchingSelector(containers, service.Spec.Selector)

	if matchingContainer == nil {
		return errors.New("no container was found matching the service selector")
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

	err = adapter.converter.ConvertServiceSpecIntoContainerConfiguration(service.Spec, &cfg)
	if err != nil {
		return fmt.Errorf("unable to convert service spec into container configuration: %w", err)
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
