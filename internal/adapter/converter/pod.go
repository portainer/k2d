package converter

import (
	"fmt"
	"path"
	"path/filepath"
	"strconv"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/go-connections/nat"
	"github.com/portainer/k2d/internal/adapter/naming"
	k2dtypes "github.com/portainer/k2d/internal/adapter/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubernetes/pkg/apis/core"
)

// ConvertContainerToPod converts a given Docker container into a Kubernetes Pod object.
// The conversion populates specific fields like TypeMeta, ObjectMeta, PodSpec, and PodStatus.
// The function currently only supports partial conversion.
//
// Parameters:
// - container: A Docker container object that will be converted into a Kubernetes Pod.
//
// Behavior:
//   - Populates the 'TypeMeta' and 'ObjectMeta' fields of the Pod object from the Docker container's metadata.
//   - Creates a single-container PodSpec based on the Docker container's image and name.
//   - Sets the Pod's status based on the Docker container's state. If the Docker container is running,
//     the Pod's phase is set to 'Running', and the container status is marked as 'Ready'. Otherwise,
//     the Pod's phase is set to 'Unknown'.
//
// Returns:
// - A Kubernetes Pod object derived from the Docker container.
func (converter *DockerAPIConverter) ConvertContainerToPod(container types.Container) core.Pod {
	containerName := container.Labels[k2dtypes.WorkloadNameLabelKey]
	containerState := container.State

	pod := core.Pod{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Pod",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:              containerName,
			CreationTimestamp: metav1.NewTime(time.Unix(container.Created, 0)),
			Namespace:         container.Labels[k2dtypes.NamespaceLabelKey],
			Annotations: map[string]string{
				"kubectl.kubernetes.io/last-applied-configuration": container.Labels[k2dtypes.WorkloadLastAppliedConfigLabelKey],
			},
		},
		Spec: core.PodSpec{
			Containers: []core.Container{
				{
					Name:  containerName,
					Image: container.Image,
				},
			},
		},
		Status: core.PodStatus{
			ContainerStatuses: []core.ContainerStatus{
				{
					Name:         containerName,
					ContainerID:  container.ID,
					RestartCount: 0,
				},
			},
		},
	}

	if containerState == "running" {
		ready := true

		pod.Status.Phase = core.PodRunning

		pod.Status.ContainerStatuses[0].Ready = ready
		pod.Status.ContainerStatuses[0].Started = &ready

		pod.Status.ContainerStatuses[0].State.Running = &core.ContainerStateRunning{
			StartedAt: metav1.NewTime(time.Unix(container.Created, 0)),
		}
	} else {
		pod.Status.Phase = core.PodUnknown
	}

	return pod
}

// ConvertPodSpecToContainerConfiguration converts a Kubernetes PodSpec into a Docker ContainerConfiguration.
//
// This function takes a PodSpec (`spec`), the namespace where the pod is to be created (`namespace`),
// and a set of labels (`labels`) as arguments. It returns a struct `ContainerConfiguration` which contains
// configurations to be used for creating a Docker container, and an error if any occurs.
//
// The function assumes the PodSpec contains at least one container specification. It only uses the first
// container in the list (`spec.Containers[0]`) for conversion.
//  1. It initializes the Docker container configuration with the image, labels, and environment variables
//     related to the Kubernetes server.
//  2. It sets additional host mappings to resolve the kubernetes service within the Docker container.
//  3. It associates the Service Account token and CA certificate with the Docker container.
//  4. It configures port mappings based on the Kubernetes container ports.
//  5. It sets environment variables based on the Kubernetes container environment settings.
//  6. It sets the container's command and arguments if they are specified in the PodSpec.
//  7. It sets the container's restart policy based on the Kubernetes Pod's restart policy.
//  8. It sets the container and host-level security context based on the PodSpec.
//  9. It sets resource requirements (CPU, memory limits, etc.) based on the Kubernetes container resources.
//  10. It configures volume mounts for the container based on the Kubernetes volume specifications.
//  11. Finally, it sets the network settings for the container, using a network name retrieved from the labels.
//
// If any of these steps fails, an error is returned.
func (converter *DockerAPIConverter) ConvertPodSpecToContainerConfiguration(spec core.PodSpec, namespace string, labels map[string]string) (ContainerConfiguration, error) {
	containerSpec := spec.Containers[0]

	containerConfig := &container.Config{
		Image:  containerSpec.Image,
		Labels: labels,
		Env: []string{
			fmt.Sprintf("KUBERNETES_SERVICE_HOST=%s", converter.k2dServerConfiguration.ServerIpAddr),
			fmt.Sprintf("KUBERNETES_SERVICE_PORT=%d", converter.k2dServerConfiguration.ServerPort),
		},
	}

	hostConfig := &container.HostConfig{
		ExtraHosts: []string{
			fmt.Sprintf("kubernetes.default.svc:%s", converter.k2dServerConfiguration.ServerIpAddr),
		},
	}

	if err := converter.SetServiceAccountTokenAndCACert(hostConfig); err != nil {
		return ContainerConfiguration{}, err
	}

	if err := converter.setHostPorts(containerConfig, hostConfig, containerSpec.Ports); err != nil {
		return ContainerConfiguration{}, err
	}

	if err := converter.setEnvVars(namespace, containerConfig, containerSpec.Env, containerSpec.EnvFrom); err != nil {
		return ContainerConfiguration{}, err
	}

	setCommandAndArgs(containerConfig, containerSpec.Command, containerSpec.Args)
	setRestartPolicy(hostConfig, spec.RestartPolicy)
	setSecurityContext(containerConfig, hostConfig, spec.SecurityContext, containerSpec.SecurityContext)
	converter.setResourceRequirements(hostConfig, containerSpec.Resources)

	if err := converter.setVolumeMounts(namespace, hostConfig, spec.Volumes, containerSpec.VolumeMounts); err != nil {
		return ContainerConfiguration{}, err
	}

	networkName := labels[k2dtypes.NetworkNameLabelKey]
	return ContainerConfiguration{
		ContainerConfig: containerConfig,
		HostConfig:      hostConfig,
		NetworkConfig: &network.NetworkingConfig{
			EndpointsConfig: map[string]*network.EndpointSettings{
				networkName: {},
			},
		},
	}, nil
}

// setResourceRequirements configures the Docker container's resource constraints based on the provided core.ResourceRequirements.
// It receives a Docker HostConfig and a Kubernetes ResourceRequirements.
func (converter *DockerAPIConverter) setResourceRequirements(hostConfig *container.HostConfig, resources core.ResourceRequirements) {
	resourceRequirements := container.Resources{}
	if resources.Requests != nil {
		for resourceName, quantity := range resources.Requests {
			switch resourceName {
			case core.ResourceCPU:
				resourceRequirements.CPUShares = int64(quantity.MilliValue())
			case core.ResourceMemory:
				resourceRequirements.MemoryReservation = int64(quantity.Value())
			}
		}
	}

	if resources.Limits != nil {
		for resourceName, quantity := range resources.Limits {
			switch resourceName {
			case core.ResourceCPU:
				resourceRequirements.NanoCPUs = int64(quantity.MilliValue()) * 1000000
			case core.ResourceMemory:
				resourceRequirements.Memory = int64(quantity.Value())
			}
		}
	}

	hostConfig.Resources = resourceRequirements
}

// SetServiceAccountTokenAndCACert configures the Docker container to have access to the service account token
// and CA certificate stored in a Kubernetes Secret. The function performs the following steps:
//  1. Fetches the service account Secret from Kubernetes using the provided secretStore.
//  2. Obtains the filesystem bind mappings for the Secret using the secretStore's GetSecretBinds method.
//  3. Modifies the hostConfig's Binds field to include the service account token and CA certificate by
//     mapping the host file system paths to the container's "/var/run/secrets/kubernetes.io/serviceaccount/" directory.
//
// Parameters:
//   - hostConfig: The Docker container's host configuration that will be modified to include the service
//     account token and CA certificate binds.
//
// It returns an error if any occurred fetching the Secret or obtaining the bind mappings fails.
func (converter *DockerAPIConverter) SetServiceAccountTokenAndCACert(hostConfig *container.HostConfig) error {
	secret, err := converter.secretStore.GetSecret(k2dtypes.K2dServiceAccountSecretName, k2dtypes.K2DNamespaceName)
	if err != nil {
		return fmt.Errorf("unable to get secret %s: %w", k2dtypes.K2dServiceAccountSecretName, err)
	}

	binds, err := converter.secretStore.GetSecretBinds(secret)
	if err != nil {
		return fmt.Errorf("unable to get binds for secrets %s: %w", k2dtypes.K2dServiceAccountSecretName, err)
	}

	for containerBind, hostBind := range binds {
		bind := fmt.Sprintf("%s:%s", hostBind, path.Join("/var/run/secrets/kubernetes.io/serviceaccount/", containerBind))
		hostConfig.Binds = append(hostConfig.Binds, bind)
	}

	return nil
}

// setHostPorts configures the Docker container's ports based on the provided core.ContainerPort slices (coming from the pod specs).
// It iterates through the ports and sets both the container's exposed ports (inside the container) and
// the host's port bindings (on the host machine). Ports are mapped only if the HostPort is not zero.
// The mappings are applied to the provided containerConfig and hostConfig.
// It returns an error if any occurred during the port conversion or mapping process.
func (converter *DockerAPIConverter) setHostPorts(containerConfig *container.Config, hostConfig *container.HostConfig, ports []core.ContainerPort) error {
	containerPortMaps := nat.PortMap{}
	containerExposedPorts := nat.PortSet{}

	for _, port := range ports {
		if port.HostPort != 0 {
			containerPort, err := nat.NewPort(string(port.Protocol), strconv.Itoa(int(port.ContainerPort)))
			if err != nil {
				return err
			}

			hostBinding := nat.PortBinding{
				HostIP:   "0.0.0.0",
				HostPort: strconv.Itoa(int(port.HostPort)),
			}

			containerPortMaps[containerPort] = []nat.PortBinding{hostBinding}
			containerExposedPorts[containerPort] = struct{}{}
		}
	}

	containerConfig.ExposedPorts = containerExposedPorts
	hostConfig.PortBindings = containerPortMaps

	return nil
}

// setEnvVars handles setting the environment variables for the Docker container configuration.
// It receives a pointer to the container configuration and an array of Kubernetes environment variables.
// It returns an error if the setting of environment variables fails.

// setEnvVars configures the environment variables for the Docker container based on Kubernetes EnvVar and EnvFromSource objects.
//
// The function receives the Kubernetes namespace (`namespace`), a pointer to the Docker container configuration (`containerConfig`),
// an array of Kubernetes EnvVar (`envs`), and an array of Kubernetes EnvFromSource (`envFrom`).
//
// It performs the following tasks:
// 1. Iterates over each EnvVar in `envs`:
//   - If EnvVar has a `ValueFrom` field, it calls `handleValueFromEnvVars` to handle the logic for populating the environment variable.
//   - Otherwise, it directly sets the environment variable in the Docker container configuration.
//
// 2. Iterates over each EnvFromSource in `envFrom`:
//   - Calls `handleValueFromEnvFromSource` to populate the environment variables based on the EnvFromSource settings.
//
// The function returns an error if any of the steps to set the environment variables fail.
func (converter *DockerAPIConverter) setEnvVars(namespace string, containerConfig *container.Config, envs []core.EnvVar, envFrom []core.EnvFromSource) error {
	for _, env := range envs {

		if env.ValueFrom != nil {
			if err := converter.handleValueFromEnvVars(namespace, containerConfig, env); err != nil {
				return err
			}
		} else {
			containerConfig.Env = append(containerConfig.Env, fmt.Sprintf("%s=%s", env.Name, env.Value))
		}
	}

	for _, env := range envFrom {
		if err := converter.handleValueFromEnvFromSource(namespace, containerConfig, env); err != nil {
			return err
		}
	}

	return nil
}

// handleValueFromEnvFromSource populates the environment variables of a Docker container configuration based on a Kubernetes EnvFromSource object.
// The function will do a lookup for ConfigMaps and Secrets within a specified Kubernetes namespace.
//
// The function takes three parameters:
// - namespace: the Kubernetes namespace where the ConfigMap or Secret is located.
// - containerConfig: a pointer to a Docker container configuration object where the environment variables will be populated.
// - env: a Kubernetes EnvFromSource object that specifies the source of the environment variables.
//
// The function performs the following actions:
// 1. If the EnvFromSource object has a ConfigMapRef, the function retrieves the ConfigMap from the given namespace using `configMapStore.GetConfigMap()`.
//   - If successful, it adds each key-value pair in the ConfigMap data as an environment variable in the Docker container configuration.
//   - Returns an error if the ConfigMap retrieval fails.
//
// 2. If the EnvFromSource object has a SecretRef, the function retrieves the Secret from the given namespace using `secretStore.GetSecret()`.
//   - If successful, it adds each key-value pair in the Secret data as an environment variable in the Docker container configuration.
//   - Returns an error if the Secret retrieval fails.
//
// The function returns nil if it successfully populates the environment variables, or an error if any step fails.
func (converter *DockerAPIConverter) handleValueFromEnvFromSource(namespace string, containerConfig *container.Config, env core.EnvFromSource) error {
	if env.ConfigMapRef != nil {
		configMap, err := converter.configMapStore.GetConfigMap(env.ConfigMapRef.Name, namespace)
		if err != nil {
			return fmt.Errorf("unable to get configmap %s: %w", env.ConfigMapRef.Name, err)
		}

		for key, value := range configMap.Data {
			containerConfig.Env = append(containerConfig.Env, fmt.Sprintf("%s=%s", key, value))
		}
	} else if env.SecretRef != nil {
		secret, err := converter.secretStore.GetSecret(env.SecretRef.Name, namespace)
		if err != nil {
			return fmt.Errorf("unable to get secret %s: %w", env.SecretRef.Name, err)
		}

		for key, value := range secret.Data {
			containerConfig.Env = append(containerConfig.Env, fmt.Sprintf("%s=%s", key, value))
		}
	}

	return nil
}

// handleValueFromEnvVars populates specific environment variables in a Docker container configuration based on ConfigMap or Secret references in a Kubernetes EnvVar object.
//
// Parameters:
// - namespace: The Kubernetes namespace where the ConfigMap or Secret resides.
// - containerConfig: A pointer to a Docker container configuration where the environment variable will be set.
// - env: A Kubernetes EnvVar object that may contain ValueFrom references to ConfigMaps or Secrets.
//
// The function performs the following actions:
// 1. If the EnvVar object has a ConfigMapKeyRef, it uses `configMapStore.GetConfigMap()` to fetch the ConfigMap by name from the specified namespace.
//   - If successful, the function fetches the value using the Key provided in ConfigMapKeyRef and sets it as an environment variable in the Docker container configuration.
//   - Returns an error if it fails to retrieve the ConfigMap.
//
// 2. If the EnvVar object has a SecretKeyRef, it uses `secretStore.GetSecret()` to fetch the Secret by name from the specified namespace.
//   - If successful, the function fetches the value using the Key provided in SecretKeyRef and sets it as an environment variable in the Docker container configuration.
//   - Returns an error if it fails to retrieve the Secret.
//
// The function returns nil upon successful population of the environment variables or an error if any step fails.
func (converter *DockerAPIConverter) handleValueFromEnvVars(namespace string, containerConfig *container.Config, env core.EnvVar) error {
	if env.ValueFrom.ConfigMapKeyRef != nil {
		configMap, err := converter.configMapStore.GetConfigMap(env.ValueFrom.ConfigMapKeyRef.Name, namespace)
		if err != nil {
			return fmt.Errorf("unable to get configmap %s: %w", env.ValueFrom.ConfigMapKeyRef.Name, err)
		}

		containerConfig.Env = append(containerConfig.Env, fmt.Sprintf("%s=%s", env.Name, configMap.Data[env.ValueFrom.ConfigMapKeyRef.Key]))
	} else if env.ValueFrom.SecretKeyRef != nil {
		secret, err := converter.secretStore.GetSecret(env.ValueFrom.SecretKeyRef.Name, namespace)
		if err != nil {
			return fmt.Errorf("unable to get secret %s: %w", env.ValueFrom.SecretKeyRef.Name, err)
		}

		containerConfig.Env = append(containerConfig.Env, fmt.Sprintf("%s=%s", env.Name, secret.Data[env.ValueFrom.SecretKeyRef.Key]))
	}
	return nil
}

// setRestartPolicy sets the Docker container's restart policy according to the Kubernetes pod's restart policy.
// It receives a pointer to the host configuration and the Kubernetes pod's restart policy.
func setRestartPolicy(hostConfig *container.HostConfig, restartPolicy core.RestartPolicy) {
	switch restartPolicy {
	case "OnFailure":
		hostConfig.RestartPolicy = container.RestartPolicy{Name: "on-failure"}
	case "Never":
		hostConfig.RestartPolicy = container.RestartPolicy{Name: "no"}
	default:
		hostConfig.RestartPolicy = container.RestartPolicy{Name: "always"}
	}
}

// setCommandAndArgs configures the entrypoint and command arguments for a given Docker container configuration.
// If the 'command' slice is non-empty, it is set as the container's entrypoint.
// If the 'args' slice is non-empty, it is set as the container's command arguments.
func setCommandAndArgs(containerConfig *container.Config, command []string, args []string) {
	if len(command) > 0 {
		containerConfig.Entrypoint = command
	}

	if len(args) > 0 {
		containerConfig.Cmd = args
	}
}

// setSecurityContext sets the user and group ID in the Docker container configuration based on the provided
// Kubernetes PodSecurityContext.
// If no security context is provided, the function does not modify the container configuration.
func setSecurityContext(config *container.Config, hostConfig *container.HostConfig, podSecurityContext *core.PodSecurityContext, containerSecurityContext *core.SecurityContext) {
	if podSecurityContext == nil {
		return
	}

	if podSecurityContext.RunAsUser != nil && podSecurityContext.RunAsGroup != nil {
		config.User = fmt.Sprintf("%d:%d", *podSecurityContext.RunAsUser, *podSecurityContext.RunAsGroup)
	}

	if containerSecurityContext == nil {
		return
	}

	if containerSecurityContext.Privileged != nil {
		hostConfig.Privileged = *containerSecurityContext.Privileged
	}
}

// setVolumeMounts manages volume mounts for the Docker container.
// It receives a pointer to the host configuration, an array of Kubernetes volumes, and an array of Kubernetes volume mounts.
// It returns an error if the handling of volume mounts fails.
func (converter *DockerAPIConverter) setVolumeMounts(namespace string, hostConfig *container.HostConfig, volumes []core.Volume, volumeMounts []core.VolumeMount) error {
	for _, volume := range volumes {
		for _, volumeMount := range volumeMounts {
			if volumeMount.Name == volume.Name {
				if err := converter.handleVolumeSource(namespace, hostConfig, volume, volumeMount); err != nil {
					return err
				}
				break
			}
		}
	}
	return nil
}

// handleVolumeSource configures the Docker host configuration's volume bindings based on a Kubernetes VolumeSource.
// The VolumeSource can be of type ConfigMap, Secret, or HostPath.
//
// For PersistentVolumeClaim:
// The function uses the volume name and namespace to generate a unique name for the volume.
//
// Parameters:
// - namespace:    The Kubernetes namespace where the volume resources (ConfigMap or Secret) are located.
// - hostConfig:   A pointer to the Docker host configuration to which the volume bindings will be appended.
// - volume:       A Kubernetes Volume object describing the source of the volume.
// - volumeMount:  A Kubernetes VolumeMount object containing additional specifications for mounting the volume.
//
// Behavior:
// - For ConfigMap and Secret:
//  1. Retrieves the resource (ConfigMap or Secret) from the store based on the volume source.
//  2. Utilizes the store's specific implementation to generate a list of filesystem binds.
//  3. Appends these binds to the 'Binds' field of the Docker host configuration.
//     - For HostPath:
//     Directly appends a bind between the HostPath and the volume mount path to the Docker host configuration.
//
// Returns:
// - An error if retrieval of the ConfigMap or Secret fails or if bind generation encounters issues.
// - Nil if the volume bindings are successfully appended to the Docker host configuration.
func (converter *DockerAPIConverter) handleVolumeSource(namespace string, hostConfig *container.HostConfig, volume core.Volume, volumeMount core.VolumeMount) error {
	if volume.VolumeSource.ConfigMap != nil {
		configMap, err := converter.configMapStore.GetConfigMap(volume.VolumeSource.ConfigMap.Name, namespace)
		if err != nil {
			return fmt.Errorf("unable to get configmap %s: %w", volume.VolumeSource.ConfigMap.Name, err)
		}

		binds, err := converter.configMapStore.GetConfigMapBinds(configMap)
		if err != nil {
			return fmt.Errorf("unable to get binds for configmap %s: %w", volume.VolumeSource.ConfigMap.Name, err)
		}

		handleStoreBinds(hostConfig, binds, volumeMount.MountPath)
	} else if volume.VolumeSource.Secret != nil {
		secret, err := converter.secretStore.GetSecret(volume.VolumeSource.Secret.SecretName, namespace)
		if err != nil {
			return fmt.Errorf("unable to get secret %s: %w", volume.VolumeSource.Secret.SecretName, err)
		}

		binds, err := converter.secretStore.GetSecretBinds(secret)
		if err != nil {
			return fmt.Errorf("unable to get binds for secrets %s: %w", volume.VolumeSource.ConfigMap.Name, err)
		}

		handleStoreBinds(hostConfig, binds, volumeMount.MountPath)
	} else if volume.HostPath != nil {
		bind := fmt.Sprintf("%s:%s", volume.HostPath.Path, volumeMount.MountPath)
		hostConfig.Binds = append(hostConfig.Binds, bind)
	} else if volume.VolumeSource.PersistentVolumeClaim != nil {
		volumeName := naming.BuildPersistentVolumeName(volume.VolumeSource.PersistentVolumeClaim.ClaimName, namespace)
		bind := fmt.Sprintf("%s:%s", volumeName, volumeMount.MountPath)
		hostConfig.Binds = append(hostConfig.Binds, bind)
	}
	return nil
}

// handleStoreBinds constructs bind mounts for Docker containers based on given host paths and container paths.
// It appends these binds to the Binds field in the given HostConfig.
//
// The function considers two scenarios:
// 1. Single-file ConfigMap:
//   - If there is only one file in the configmap, the function will attempt to mount it directly to the file in the container if
//     mountPath has a file extension. Otherwise, it will mount to a folder.
//
// 2. Multiple-file ConfigMap:
//   - If there are multiple files in the configmap, the function will mount each host file to its corresponding container path.
//
// Special handling is performed when the mount path in the container is already a file (has an extension), and there is only one bind.
// In this case, the parent directory of the existing file is used as the mount path. This is to support mounting a file to a file when using the disk
// backend, and mounting a volume to the parent directory of the file when using the volume backend.
// For example, when mountPath is set to /etc/influxdb/influxdb.conf, this function needs to be able to mount the /path/to/host/influxdb.conf file
// directly to the /etc/influxdb/influxdb.conf file in the container when using the disk backend. However, when using the volume backend, the
// volume that contains the influxdb.conf file needs to be mounted to the /etc/influxdb directory in the container.
//
// Parameters:
// - hostConfig: The Docker container's host configuration where the bind mounts are appended.
// - binds: A map where the key is the container path (containerBind) and the value is the host path (hostBind).
// - mountPath: The target file in the container where the host files will be mounted.
//
// Note:
// - For disk backend, binds map entries would be like {"filename": "/path/on/host"}
// - For volume backend, binds map entries would be like {"": "volumename"}
func handleStoreBinds(hostConfig *container.HostConfig, binds map[string]string, mountPath string) {
	for containerBind, hostBind := range binds {
		bind := fmt.Sprintf("%s:%s", hostBind, path.Join(mountPath, containerBind))
		if len(binds) == 1 && filepath.Ext(mountPath) != "" {
			bind = fmt.Sprintf("%s:%s", hostBind, path.Join(filepath.Dir(mountPath), containerBind))
		}

		hostConfig.Binds = append(hostConfig.Binds, bind)
	}
}
