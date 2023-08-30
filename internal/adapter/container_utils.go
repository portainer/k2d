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
			Resources:     containerDetails.HostConfig.Resources,
		},
		NetworkConfig: &network.NetworkingConfig{
			EndpointsConfig: containerDetails.NetworkSettings.Networks,
		},
	}, nil
}

// ContainerCreationOptions is a struct used to provide parameters for creating a container.
// It includes the containerName which is a string indicating the name of the container to be created,
// a PodSpec which represents the desired state of the Pod from the parent Kubernetes object,
// and labels which is a map of key-value pairs.
// It also includes a string representation of the last applied configuration of the parent Kubernetes object.
// TODO: update with namespace support
type ContainerCreationOptions struct {
	containerName            string
	namespace                string
	podSpec                  corev1.PodSpec
	labels                   map[string]string
	lastAppliedConfiguration string
}

// TODO: comment
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

// TODO: update with namespace support
// createContainerFromPodSpec creates a new Docker container from a given Kubernetes PodSpec.
// The function first initializes labels if they are not provided and adds the last applied configuration
// to the labels if it's specified. It then converts the versioned pod spec to an internal pod spec
// and serializes it to JSON to be stored as a label on the Docker container.
//
// It attempts to convert the internal PodSpec to a Docker container configuration and lists existing
// Docker containers to check if a container with the specified name already exists.
//
// If a matching container is found and has the same last applied configuration, the update is skipped.
// If the existing container has a different configuration, it is removed, and its network aliases and
// service last applied configuration label are preserved if present.
//
// The function then pulls the necessary image using the registry credentials obtained for the image
// in the provided pod spec. If successful, it creates and starts the new Docker container, attaching
// it to a predefined Docker network.
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
// If there is an error at any point in the process (e.g., conversion failure, image pull failure, container removal or creation),
// the function returns the error, wrapped with a description of the step that failed.
func (adapter *KubeDockerAdapter) createContainerFromPodSpec(ctx context.Context, options ContainerCreationOptions) error {
	if options.lastAppliedConfiguration != "" {
		options.labels[k2dtypes.WorkloadLastAppliedConfigLabelKey] = options.lastAppliedConfiguration
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
	options.labels[k2dtypes.NamespaceLabelKey] = options.namespace
	options.labels[k2dtypes.WorkloadNameLabelKey] = options.containerName
	options.labels[k2dtypes.NetworkNameLabelKey] = buildNetworkName(options.namespace)

	containerCfg, err := adapter.converter.ConvertPodSpecToContainerConfiguration(internalPodSpec, options.namespace, options.labels)
	if err != nil {
		return fmt.Errorf("unable to build container configuration from pod spec: %w", err)
	}
	containerCfg.ContainerName = buildContainerName(options.containerName, options.namespace)

	existingContainer, err := adapter.getContainer(ctx, containerCfg.ContainerName)
	if err != nil {
		return fmt.Errorf("unable to inspect container: %w", err)
	}

	if existingContainer != nil {
		if options.lastAppliedConfiguration == existingContainer.Config.Labels[k2dtypes.WorkloadLastAppliedConfigLabelKey] {
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

//TODO: function comment with namespace

// DeleteContainer removes a Docker container given its ID or name.
// This function will force the removal of the container, regardless if it's running or not.
// It will log a warning if it fails to delete the container.
func (adapter *KubeDockerAdapter) DeleteContainer(ctx context.Context, containerName, namespace string) error {
	err := adapter.cli.ContainerRemove(ctx, buildContainerName(containerName, namespace), types.ContainerRemoveOptions{Force: true})
	if err != nil {
		adapter.logger.Warnf("unable to remove container: %s", err)
	}

	return nil
}

// getRegistryCredentials retrieves the registry credentials for a given image name
// within the specified pod specification. If the podSpec's ImagePullSecrets are nil,
// it returns an empty string without an error.
//
// The function first normalizes the image name by adding the "docker.io/" prefix if it
// lacks a registry domain. Then, it parses the image name to obtain the registry URL.
//
// Using the first pull secret from podSpec.ImagePullSecrets, the function fetches the registry
// secret and decodes it using the given registry URL. It constructs an authentication config
// from the obtained username and password and returns its base64-encoded JSON representation.
//
// If any step fails, an error is returned.
// TODO: update docs with namespace
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

	adapter.logger.Debugw("retrieving private registry credentials",
		"container_image", imageName,
		"registry", registryURL,
	)

	// We only support a single image pull secret for now
	pullSecret := podSpec.ImagePullSecrets[0]

	registrySecret, err := adapter.registrySecretStore.GetSecret(pullSecret.Name, namespace)
	if err != nil {
		return "", fmt.Errorf("unable to get registry secret: %w", err)
	}

	username, password, err := k8s.GetRegistryAuthFromSecret(registrySecret, registryURL)
	if err != nil {
		return "", fmt.Errorf("unable to decode registry secret: %w", err)
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

	networkName := buildNetworkName(k2dtypes.K2DNamespaceName)
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
