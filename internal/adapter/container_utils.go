package adapter

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/docker/distribution/reference"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/registry"
	"github.com/docker/docker/errdefs"
	"github.com/docker/go-connections/nat"
	"github.com/portainer/k2d/internal/adapter/converter"
	"github.com/portainer/k2d/internal/adapter/naming"
	k2dtypes "github.com/portainer/k2d/internal/adapter/types"
	"github.com/portainer/k2d/internal/k8s"
	"github.com/portainer/k2d/pkg/maputils"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/kubernetes/pkg/apis/core"
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

// reCreateContainerWithNewConfiguration replaces an existing Docker container with a new one that has an updated configuration.
// The function performs the following steps:
// 1. Stops the existing container by its containerID.
// 2. Creates a new container using the newContainerCfg with a temporary name.
// 3. Starts the newly created container.
// 4. If the new container starts successfully, removes the old container.
// 5. Renames the new container to have the original name as specified in newContainerCfg.
// If any of the steps fail:
// - When failing to create a new container, the function attempts to restart the old container.
// - When failing to start the new container, the old container is removed, and the new container is left in a created state and renamed to the original name for inspection.
//
// Parameters:
// - ctx: Context used for cancellation or timeouts.
// - containerID: The ID of the existing Docker container to be replaced.
// - newContainerCfg: The new container configuration.
//
// Returns:
// - An error if any of the steps fail.
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
		// If the new container fails to start, remove the old container and leave the new container in the created state.
		// This way, the container can be inspected to see what went wrong.
		removeErr := adapter.cli.ContainerRemove(ctx, containerID, types.ContainerRemoveOptions{})
		if removeErr != nil {
			return fmt.Errorf("unable to remove the old container after failed start: %w", removeErr)
		}

		// Rename the new container to the original name
		// This way, the container with the known name can be inspected to see what went wrong.
		renameErr := adapter.cli.ContainerRename(ctx, containerCreateResponse.ID, newContainerCfg.ContainerName)
		if renameErr != nil {
			return fmt.Errorf("unable to rename container for the new container: %w", renameErr)
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

	adapter.logger.Debugf("container details: %+v", containerDetails)

	containerConfiguration := converter.ContainerConfiguration{
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
			Resources:     containerDetails.HostConfig.Resources,
		},
		NetworkConfig: &network.NetworkingConfig{
			EndpointsConfig: containerDetails.NetworkSettings.Networks,
		},
	}

	adapter.logger.Debugf("container configuration: %+v", containerConfiguration)

	// We set the ulimits to nil because Podman is not able to create the container when inheriting the ulimits
	// from the previous container configuration.
	containerConfiguration.HostConfig.Resources.Ulimits = nil

	return containerConfiguration, nil
}

// ContainerCreationOptions serves as a parameter object for container creation operations.
// The struct encapsulates various attributes required for configuring a container, as described below:
//
//   - containerName: Specifies the name of the container to be created.
//   - labels: A map representing key-value pairs of labels that will be attached to the container.
//     These labels are useful for organizational and operational tasks like filtering and grouping.
//   - lastAppliedConfiguration: A string containing the serialized state of the last applied configuration
//     for the parent Kubernetes object. This is used to manage updates and rollbacks.
//   - namespace: Indicates the Kubernetes namespace within which the container should reside.
//     This is used to ensure that the container is created in the correct network.
//   - podSpec: Holds the corev1.PodSpec object representing the desired state of the associated Pod.
//     This includes configurations like the container image, environment variables, and volume mounts.
type ContainerCreationOptions struct {
	containerName            string
	labels                   map[string]string
	lastAppliedConfiguration string
	namespace                string
	podSpec                  corev1.PodSpec
}

// getContainer inspects the specified container and returns its details in the form of a pointer to a types.ContainerJSON object.
// This is a handy wrapper around the Docker SDK's ContainerInspect function that allows us to easily check if a container exists.
// The function takes two parameters:
//
// - ctx: The context within which this operation should be executed, useful for timeout and cancellation.
// - containerName: The name (or ID) of the container that needs to be inspected.
//
// The function has two return values:
//
//   - *types.ContainerJSON: A pointer to the object that holds the inspected details of the container.
//     This pointer will be nil if the container with the specified name is not found.
//   - error: An error object that will be returned in case of failure to inspect the container.
//     It will wrap the original error message with additional context, if any.
func (adapter *KubeDockerAdapter) getContainer(ctx context.Context, containerName string) (*types.ContainerJSON, error) {
	containerDetails, err := adapter.cli.ContainerInspect(ctx, containerName)
	if err != nil {
		if errdefs.IsNotFound(err) {
			return nil, nil
		}

		return nil, fmt.Errorf("unable to inspect container: %w", err)
	}

	return &containerDetails, nil
}

// createContainerFromPodSpec orchestrates the creation of a Docker container based on a given Kubernetes PodSpec.
// The function goes through several key steps in the container creation lifecycle:
//
//  1. Initializes and updates container labels using the last applied configuration if provided.
//  2. Converts the provided Kubernetes PodSpec into an internal PodSpec, which is then serialized to JSON.
//     This serialized form is stored as a label on the Docker container for future reference.
//  3. Constructs a Docker container configuration from the internal PodSpec.
//  4. Checks for an existing Docker container with the same name:
//     - If found with an identical last applied configuration, skips the update.
//     - If found but but with a different last applied configuration, removes the existing container.
//  5. Pulls the necessary Docker image using registry credentials from the Kubernetes PodSpec.
//  6. Creates and starts the Docker container.
//
// Parameters:
// - ctx: The operational context within which the function runs. Used for timeouts and cancellation signals.
// - options: A ContainerCreationOptions struct containing the necessary parameters for container creation.
//   - containerName: Specifies the name of the Docker container to create.
//   - labels: A map of labels to attach to the Docker container.
//   - lastAppliedConfiguration: Stores the last configuration applied to the parent Kubernetes object.
//     This is saved as a label on the Docker container.
//   - namespace: Used to determine the network in which the container should be created.
//   - podSpec: The Kubernetes PodSpec that serves as the template for the Docker container.
//
// Returns:
//   - If any step in the container creation process fails (such as PodSpec conversion, image pull, or container creation),
//     the function returns an error wrapped with a description of the failed step.
func (adapter *KubeDockerAdapter) createContainerFromPodSpec(ctx context.Context, options ContainerCreationOptions) error {
	if options.lastAppliedConfiguration != "" {
		options.labels[k2dtypes.LastAppliedConfigLabelKey] = options.lastAppliedConfiguration
	}

	internalPodSpec := core.PodSpec{}
	err := adapter.ConvertK8SResource(&options.podSpec, &internalPodSpec)
	if err != nil {
		return fmt.Errorf("unable to convert versioned pod spec to internal pod spec: %w", err)
	}

	internalPodSpecData, err := json.Marshal(internalPodSpec)
	if err != nil {
		return fmt.Errorf("unable to marshal internal pod spec: %w", err)
	}
	options.labels[k2dtypes.PodLastAppliedConfigLabelKey] = string(internalPodSpecData)
	options.labels[k2dtypes.NamespaceNameLabelKey] = options.namespace
	options.labels[k2dtypes.WorkloadNameLabelKey] = options.containerName
	options.labels[k2dtypes.NetworkNameLabelKey] = naming.BuildNetworkName(options.namespace)

	containerCfg, err := adapter.converter.ConvertPodSpecToContainerConfiguration(internalPodSpec, options.namespace, options.labels)
	if err != nil {
		return fmt.Errorf("unable to build container configuration from pod spec: %w", err)
	}
	containerCfg.ContainerName = naming.BuildContainerName(options.containerName, options.namespace)

	existingContainer, err := adapter.getContainer(ctx, containerCfg.ContainerName)
	if err != nil {
		return fmt.Errorf("unable to inspect container: %w", err)
	}

	if existingContainer != nil {
		if options.lastAppliedConfiguration == existingContainer.Config.Labels[k2dtypes.LastAppliedConfigLabelKey] {
			adapter.logger.Infof("container with the name %s already exists with the same configuration. The update will be skipped", containerCfg.ContainerName)
			return nil
		}

		adapter.logger.Infof("container with the name %s already exists with a different configuration. The container will be recreated", containerCfg.ContainerName)

		if existingContainer.Config.Labels[k2dtypes.ServiceLastAppliedConfigLabelKey] != "" {
			options.labels[k2dtypes.ServiceLastAppliedConfigLabelKey] = existingContainer.Config.Labels[k2dtypes.ServiceLastAppliedConfigLabelKey]
		}

		err := adapter.cli.ContainerRemove(ctx, existingContainer.ID, types.ContainerRemoveOptions{Force: true})
		if err != nil {
			return fmt.Errorf("unable to remove container: %w", err)
		}
	}

	registryAuth, err := adapter.getRegistryCredentials(options.podSpec, options.namespace, containerCfg.ContainerConfig.Image)
	if err != nil {
		return fmt.Errorf("unable to get registry credentials: %w", err)
	}

	out, err := adapter.cli.ImagePull(ctx, containerCfg.ContainerConfig.Image, types.ImagePullOptions{
		RegistryAuth: registryAuth,
	})
	if err != nil {
		return fmt.Errorf("unable to pull %s image: %w", containerCfg.ContainerConfig.Image, err)
	}
	defer out.Close()

	io.Copy(os.Stdout, out)

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

// DeleteContainer attempts to remove a Docker container based on its name and associated namespace.
// The container name is fully qualified by appending the namespace to it using the buildContainerName function.
// This function forcefully removes the container, regardless of whether it is running or not.
//
// The function performs the following steps:
// 1. Constructs the fully qualified container name by appending the namespace to the provided container name.
// 2. Calls the Docker API's ContainerRemove method to forcefully remove the container.
//
// If there is an error during the container removal process, a warning message will be logged.
//
// Parameters:
// - ctx: The context within which the function operates, useful for timeout and cancellation signals.
// - containerName: The base name of the Docker container to be removed.
// - namespace: The Kubernetes namespace associated with the container, used for constructing the fully qualified container name.
//
// Returns:
//   - This function does not return any value or error. Failures in container removal are only logged as warnings.
//     This is because the container may not exist anymore, and the function should not fail in that case.
func (adapter *KubeDockerAdapter) DeleteContainer(ctx context.Context, containerName, namespace string) {
	containerName = naming.BuildContainerName(containerName, namespace)

	err := adapter.cli.ContainerRemove(ctx, containerName, types.ContainerRemoveOptions{Force: true})
	if err != nil {
		adapter.logger.Warnf("unable to remove container: %s", err)
	}
}

// getRegistryCredentials attempts to retrieve the Docker registry credentials for a given image name
// within the specified Kubernetes PodSpec and namespace.
//
// The function performs the following steps:
// 1. Checks if podSpec.ImagePullSecrets is nil. If it is, the function returns an empty string without an error.
// 2. Normalizes the image name by prefixing it with "docker.io/" if it lacks a registry domain.
// 3. Parses the normalized image name to extract the registry URL.
// 4. Logs an info message indicating the retrieval of registry credentials.
// 5. Fetches the first pull secret from podSpec.ImagePullSecrets and retrieves the associated Kubernetes Secret.
// 6. Decodes the Kubernetes Secret to get the username and password for the Docker registry.
// 7. Constructs a Docker AuthConfig structure using the obtained username and password.
// 8. Serializes the AuthConfig to JSON and encodes it to a base64 string.
//
// Parameters:
// - podSpec: The Kubernetes PodSpec containing the ImagePullSecrets.
// - namespace: The Kubernetes namespace in which to look for the ImagePullSecret.
// - imageName: The name of the Docker image for which to retrieve registry credentials.
//
// Returns:
//   - A base64-encoded JSON string containing the Docker registry credentials, or an empty string if ImagePullSecrets is nil.
//   - An error if any step in the process fails, such as parsing the image name, fetching the Kubernetes Secret, decoding the Secret,
//     or serializing the AuthConfig.
//
// Note: Currently, the function only supports a single ImagePullSecret.
func (adapter *KubeDockerAdapter) getRegistryCredentials(podSpec corev1.PodSpec, namespace, imageName string) (string, error) {
	if podSpec.ImagePullSecrets == nil {
		return "", nil
	}

	if !strings.Contains(imageName, "/") || !strings.Contains(strings.Split(imageName, "/")[0], ".") {
		imageName = "docker.io/" + imageName
	}

	parsed, err := reference.ParseNamed(imageName)
	if err != nil {
		return "", fmt.Errorf("unable to parse image name: %w", err)
	}

	registryURL := reference.Domain(parsed)

	adapter.logger.Infow("retrieving private registry credentials",
		"container_image", imageName,
		"registry", registryURL,
	)

	pullSecret := podSpec.ImagePullSecrets[0]

	registrySecret, err := adapter.registrySecretStore.GetSecret(pullSecret.Name, namespace)
	if err != nil {
		return "", fmt.Errorf("unable to get registry secret %s: %w", pullSecret.Name, err)
	}

	username, password, err := k8s.GetRegistryAuthFromSecret(registrySecret, registryURL)
	if err != nil {
		return "", fmt.Errorf("unable to decode registry secret %s: %w", pullSecret.Name, err)
	}

	authConfig := registry.AuthConfig{
		Username: username,
		Password: password,
	}

	encodedAuthConfig, err := json.Marshal(authConfig)
	if err != nil {
		return "", fmt.Errorf("unable to marshal auth config: %w", err)
	}

	return base64.URLEncoding.EncodeToString(encodedAuthConfig), nil
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
		ExtraHosts: []string{
			fmt.Sprintf("kubernetes.default.svc:%s", adapter.k2dServerConfiguration.ServerIpAddr),
		},
		RestartPolicy: container.RestartPolicy{
			Name: "always",
		},
	}

	if err := adapter.converter.SetServiceAccountTokenAndCACert(hostConfig); err != nil {
		return fmt.Errorf("unable to set service account token and CA cert: %w", err)
	}

	networkName := naming.BuildNetworkName(k2dtypes.K2DNamespaceName)
	networkConfig := &network.NetworkingConfig{
		EndpointsConfig: map[string]*network.EndpointSettings{
			networkName: {},
		},
	}

	out, err := adapter.cli.ImagePull(ctx, containerConfig.Image, types.ImagePullOptions{})
	if err != nil {
		return fmt.Errorf("unable to pull %s image: %w", containerConfig.Image, err)
	}
	defer out.Close()

	io.Copy(os.Stdout, out)

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
