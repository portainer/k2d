package adapter

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/go-connections/nat"
	"github.com/portainer/k2d/internal/adapter/converter"
	k2dtypes "github.com/portainer/k2d/internal/adapter/types"
	"github.com/portainer/k2d/internal/logging"
	"github.com/portainer/k2d/pkg/maputils"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/uuid"
)

// findContainerMatchingSelector iterates over a slice of Container types, looking for a Container
// whose Labels contain a key-value pair specified in the provided selector map.
// The function returns a pointer to the first matching Container it finds.
// If no matching Container is found, the function returns nil.
func findContainerMatchingSelector(containers []types.Container, selector map[string]string) *types.Container {
	for _, container := range containers {
		for key, value := range container.Labels {
			if maputils.ContainsKeyValuePairInMap(key, value, selector) {
				return &container
			}
		}
	}

	return nil
}

// reCreateContainerWithNewConfiguration replaces an existing Docker container with a new one having updated configuration.
// The function performs the following steps:
// 1. Stops the existing container.
// 2. Creates a new container with the updated configuration using a temporary name.
// 3. Attempts to start the new container.
// 4. If the new container starts successfully, it removes the old container and renames the new container to the original name.
// In case of failure during the creation or start of the new container, it attempts to restart the old container and remove the new one.
// This way, the function ensures that a working container is always available.
// The function takes three arguments:
// - ctx is the context, used to control cancellation or timeouts for the operations.
// - containerID is the ID of the existing container that needs to be replaced.
// - newContainerCfg is the configuration for the new container.
// The function returns an error if any step in the process fails.
func (adapter *KubeDockerAdapter) reCreateContainerWithNewConfiguration(ctx context.Context, containerID string, newContainerCfg converter.ContainerConfiguration) error {
	// Define temporary container name
	tempContainerName := newContainerCfg.ContainerName + "_temp"

	// Stop the existing container
	containerStopTimeout := 3
	err := adapter.cli.ContainerStop(ctx, containerID, container.StopOptions{Timeout: &containerStopTimeout})
	if err != nil {
		return fmt.Errorf("unable to stop existing container: %w", err)
	}

	// Create a new container
	containerCreateResponse, err := adapter.cli.ContainerCreate(ctx,
		newContainerCfg.ContainerConfig,
		newContainerCfg.HostConfig,
		newContainerCfg.NetworkConfig,
		nil,
		tempContainerName,
	)
	if err != nil {
		// Attempt to start the old container again in case of failure
		if startErr := adapter.cli.ContainerStart(ctx, containerID, types.ContainerStartOptions{}); startErr != nil {
			return fmt.Errorf("unable to start the old container after failed new container creation: %w", startErr)
		}
		return fmt.Errorf("unable to create container: %w", err)
	}

	// Start the new container
	err = adapter.cli.ContainerStart(ctx, containerCreateResponse.ID, types.ContainerStartOptions{})
	if err != nil {
		// If the new container fails to start, attempt cleanup and start the old container again
		if removeErr := adapter.cli.ContainerRemove(ctx, containerCreateResponse.ID, types.ContainerRemoveOptions{}); removeErr != nil {
			return fmt.Errorf("unable to remove the newly created container after failed start: %w", removeErr)
		}

		if startErr := adapter.cli.ContainerStart(ctx, containerID, types.ContainerStartOptions{}); startErr != nil {
			return fmt.Errorf("unable to start the old container after failed new container start: %w", startErr)
		}
		return fmt.Errorf("unable to start container: %w", err)
	}

	// If the new container started successfully, remove the old container
	err = adapter.cli.ContainerRemove(ctx, containerID, types.ContainerRemoveOptions{})
	if err != nil {
		return fmt.Errorf("unable to remove old container: %w", err)
	}

	// Rename the new container to the original name
	err = adapter.cli.ContainerRename(ctx, containerCreateResponse.ID, newContainerCfg.ContainerName)
	if err != nil {
		return fmt.Errorf("unable to rename container: %w", err)
	}

	return nil
}

// buildContainerConfigurationFromExistingContainer builds a ContainerConfiguration from an existing Docker container.
// This function must be updated when adding support to new container configuration options.
func (adapter *KubeDockerAdapter) buildContainerConfigurationFromExistingContainer(ctx context.Context, containerID string) (converter.ContainerConfiguration, error) {
	containerDetails, err := adapter.cli.ContainerInspect(ctx, containerID)
	if err != nil {
		return converter.ContainerConfiguration{}, fmt.Errorf("unable to inspect container: %w", err)
	}

	return converter.ContainerConfiguration{
		ContainerName: containerDetails.Name,
		ContainerConfig: &container.Config{
			Image:        containerDetails.Image,
			Labels:       containerDetails.Config.Labels,
			ExposedPorts: nat.PortSet{},
			Env:          containerDetails.Config.Env,
			User:         containerDetails.Config.User,
		},
		HostConfig: &container.HostConfig{
			PortBindings:  nat.PortMap{},
			RestartPolicy: containerDetails.HostConfig.RestartPolicy,
			Binds:         containerDetails.HostConfig.Binds,
			ExtraHosts:    containerDetails.HostConfig.ExtraHosts,
			Privileged:    containerDetails.HostConfig.Privileged,
		},
		NetworkConfig: &network.NetworkingConfig{
			EndpointsConfig: map[string]*network.EndpointSettings{
				k2dtypes.K2DNetworkName: {},
			},
		},
	}, nil
}

// ContainerCreationOptions is a struct used to provide parameters for creating a container.
// It includes the containerName which is a string indicating the name of the container to be created,
// a PodSpec which represents the desired state of the Pod from the parent Kubernetes object,
// and labels which is a map of key-value pairs.
// It also includes a string representation of the last applied configuration of the parent Kubernetes object.
type ContainerCreationOptions struct {
	containerName            string
	podSpec                  corev1.PodSpec
	labels                   map[string]string
	lastAppliedConfiguration string
}

// createContainerFromPodSpec creates a new Docker container from a given Kubernetes PodSpec.
// The function attempts to convert the PodSpec to a Docker container configuration.
// It first checks if a container with the same name specified in the ContainerCreationOptions already exists.
// If it exists and has the same configuration, the update is skipped and the function returns.
// If it exists but has a different configuration, the old container is removed.
// The function then pulls the necessary image, and if successful, creates and starts the new Docker container.
// The created container is attached to a predefined Docker network.
//
// Parameters:
// ctx - The context within which the function works. Used for timeout and cancellation signals.
// options - ContainerCreationOptions struct, contains parameters needed for container creation:
//   - containerName: Name of the container.
//   - podSpec: Kubernetes PodSpec to base the Docker container on.
//   - labels: Map of labels to be attached to the Docker container.
//   - lastAppliedConfiguration: String representation of the last applied configuration of the parent Kubernetes object.
//     This field is stored as a label on the Docker container.
//
// If there is an error at any point in the process (conversion failure, image pull failure, etc.), the function returns the error.
func (adapter *KubeDockerAdapter) createContainerFromPodSpec(ctx context.Context, options ContainerCreationOptions) error {
	if options.labels == nil {
		options.labels = map[string]string{}
	}

	if options.lastAppliedConfiguration != "" {
		options.labels[k2dtypes.WorkloadLastAppliedConfigLabelKey] = options.lastAppliedConfiguration
	}

	containerCfg, err := adapter.converter.ConvertPodSpecToContainerConfiguration(options.podSpec, options.labels)
	if err != nil {
		return fmt.Errorf("unable to build container configuration from pod spec: %w", err)
	}
	containerCfg.ContainerName = options.containerName

	containers, err := adapter.cli.ContainerList(ctx, types.ContainerListOptions{})
	if err != nil {
		return fmt.Errorf("unable to list containers: %w", err)
	}

	for _, container := range containers {
		if container.Names[0] == "/"+containerCfg.ContainerName {
			logger := logging.LoggerFromContext(ctx)

			if options.lastAppliedConfiguration == container.Labels[k2dtypes.WorkloadLastAppliedConfigLabelKey] {
				logger.Infof("container with the name %s already exists with the same configuration. The update will be skipped", containerCfg.ContainerName)
				return nil
			}

			logger.Infof("container with the name %s already exists with a different configuration. The container will be recreated", containerCfg.ContainerName)

			err := adapter.cli.ContainerRemove(ctx, container.ID, types.ContainerRemoveOptions{Force: true})
			if err != nil {
				return fmt.Errorf("unable to remove container: %w", err)
			}
		}
	}

	out, err := adapter.cli.ImagePull(ctx, containerCfg.ContainerConfig.Image, types.ImagePullOptions{})
	if err != nil {
		return fmt.Errorf("unable to pull %s image: %w", containerCfg.ContainerConfig.Image, err)
	}
	defer out.Close()

	io.Copy(io.Discard, out)

	containerCreateResponse, err := adapter.cli.ContainerCreate(ctx,
		containerCfg.ContainerConfig,
		containerCfg.HostConfig,
		containerCfg.NetworkConfig,
		nil,
		containerCfg.ContainerName,
	)
	if err != nil {
		return fmt.Errorf("unable to create container: %w", err)
	}

	return adapter.cli.ContainerStart(ctx, containerCreateResponse.ID, types.ContainerStartOptions{})
}

// DeleteContainer removes a Docker container given its ID or name.
// This function will force the removal of the container, regardless if it's running or not.
// It will return an error if the Docker client fails to remove the container for any reason.
func (adapter *KubeDockerAdapter) DeleteContainer(ctx context.Context, containerID string) error {
	err := adapter.cli.ContainerRemove(ctx, containerID, types.ContainerRemoveOptions{Force: true})
	if err != nil {
		return fmt.Errorf("unable to remove container: %w", err)
	}

	return nil
}

// DeployPortainerEdgeAgent deploys a Portainer Edge Agent as a Docker container.
// The function first checks if a container using the Portainer Agent image already exists.
// If the container does not exist, it creates and starts a new one with the specified configurations.
//
// Parameters:
// ctx - The context within which the function works. Used for timeout and cancellation signals.
// edgeKey - The edge key for the Portainer Edge Agent.
// edgeID - The edge ID for the Portainer Edge Agent. If it's an empty string, a new UUID will be generated.
// agentVersion - The version of the Portainer Edge Agent to deploy.
//
// Returns:
// If the function fails at any point (unable to list containers, unable to pull the image, unable to create the container, or unable to start the container),
// it will return an error.
//
// If a container using the Portainer Agent image already exists, the function will log this information and return nil (indicating that no error occurred).
//
// If a container using the Portainer Agent image does not exist, the function will create and start it,
// then return nil to indicate that the process was successful.
func (adapter *KubeDockerAdapter) DeployPortainerEdgeAgent(ctx context.Context, edgeKey, edgeID, agentVersion string) error {
	containers, err := adapter.cli.ContainerList(ctx, types.ContainerListOptions{All: true})
	if err != nil {
		return fmt.Errorf("unable to list docker containers: %w", err)
	}

	for _, container := range containers {
		if strings.Contains(container.Image, "portainer/agent") {
			adapter.logger.Info("a container using the portainer/agent was found on the system, skipping creation")
			return nil
		}
	}

	if edgeID == "" {
		edgeID = string(uuid.NewUUID())
	}

	adapter.logger.Infow("deploying portainer agent container",
		"edge_key", edgeKey,
		"edge_id", edgeID,
		"agent_version", agentVersion,
	)

	containerConfig := &container.Config{
		Image: "portainer/agent:" + agentVersion,
		Env: []string{
			"EDGE=1",
			"EDGE_ID=" + edgeID,
			"EDGE_KEY=" + edgeKey,
			"EDGE_INSECURE_POLL=1",
			"EDGE_ASYNC=1",
			"KUBERNETES_POD_IP=127.0.0.1",
			"AGENT_CLUSTER_ADDR=127.0.0.1",
			"LOG_LEVEL=DEBUG",
			fmt.Sprintf("KUBERNETES_SERVICE_HOST=%s", adapter.k2dServerConfiguration.ServerIpAddr),
			fmt.Sprintf("KUBERNETES_SERVICE_PORT=%d", adapter.k2dServerConfiguration.ServerPort),
		},
	}

	hostConfig := &container.HostConfig{
		Binds: []string{
			fmt.Sprintf("%s:%s", adapter.k2dServerConfiguration.CaPath, "/var/run/secrets/kubernetes.io/serviceaccount/ca.crt"),
			fmt.Sprintf("%s:%s", adapter.k2dServerConfiguration.TokenPath, "/var/run/secrets/kubernetes.io/serviceaccount/token"),
		},
		ExtraHosts: []string{
			fmt.Sprintf("kubernetes.default.svc:%s", adapter.k2dServerConfiguration.ServerIpAddr),
		},
		RestartPolicy: container.RestartPolicy{
			Name: "always",
		},
	}

	networkConfig := &network.NetworkingConfig{
		EndpointsConfig: map[string]*network.EndpointSettings{
			k2dtypes.K2DNetworkName: {},
		},
	}

	out, err := adapter.cli.ImagePull(ctx, containerConfig.Image, types.ImagePullOptions{})
	if err != nil {
		return fmt.Errorf("unable to pull %s image: %w", containerConfig.Image, err)
	}
	defer out.Close()

	io.Copy(io.Discard, out)

	_, err = adapter.cli.ContainerCreate(ctx, containerConfig, hostConfig, networkConfig, nil, "portainer-agent")

	if err != nil {
		return fmt.Errorf("unable to create portainer agent container: %w", err)
	}

	err = adapter.cli.ContainerStart(ctx, "portainer-agent", types.ContainerStartOptions{})
	if err != nil {
		return fmt.Errorf("unable to start portainer agent container: %w", err)
	}

	return nil
}
