package adapter

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/portainer/k2d/internal/adapter/errors"
	"github.com/portainer/k2d/internal/adapter/filters"
	"github.com/portainer/k2d/internal/adapter/naming"
	k2dtypes "github.com/portainer/k2d/internal/adapter/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubernetes/pkg/apis/core"
)

// buildPodFromContainer converts a Docker container into a Kubernetes Pod object.
// The function leverages an internal converter to map the basic attributes of a container
// to a Pod. Additionally, it attempts to extract the last-applied PodSpec configuration
// (if available) from the container labels and sets it to the Pod's Spec field.
//
// Parameters:
// - container: The Docker container that needs to be converted into a Pod.
//
// Returns:
// - core.Pod: The converted Pod object.
// - error: An error object if any error occurs during the conversion.
func (adapter *KubeDockerAdapter) buildPodFromContainer(container types.Container) (core.Pod, error) {
	pod := adapter.converter.ConvertContainerToPod(container)

	if container.Labels[k2dtypes.PodLastAppliedConfigLabelKey] != "" {
		internalPodSpecData := container.Labels[k2dtypes.PodLastAppliedConfigLabelKey]
		podSpec := core.PodSpec{}

		err := json.Unmarshal([]byte(internalPodSpecData), &podSpec)
		if err != nil {
			return core.Pod{}, fmt.Errorf("unable to unmarshal pod spec: %w", err)
		}

		pod.Spec = podSpec
	}

	return pod, nil
}

// findContainerFromPodAndNamespace searches for a Docker container based on a given Pod name and namespace.
// It lists all the containers and filters them based on the Pod and namespace information.
// If the namespace is neither 'default' nor empty, it adds specific filters to pinpoint the search.
//
// Parameters:
// - ctx: The context within which the function operates.
// - podName: The name of the Pod for which to find the container.
// - namespace: The Kubernetes namespace where the Pod resides.
//
// Returns:
// - *types.Container: A pointer to the matching Docker container.
// - error: An error object if the container is not found or any other error occurs.
func (adapter *KubeDockerAdapter) findContainerFromPodAndNamespace(ctx context.Context, podName string, namespace string) (*types.Container, error) {
	var container *types.Container

	listOptions := types.ContainerListOptions{All: true}
	containerName := podName

	if !isDefaultOrEmptyNamespace(namespace) {
		listOptions.Filters = filters.ByPod(namespace, podName)
		containerName = naming.BuildContainerName(podName, namespace)
	}

	containers, err := adapter.cli.ContainerList(ctx, listOptions)
	if err != nil {
		return nil, fmt.Errorf("unable to list containers: %w", err)
	}

	for _, cntr := range containers {
		updateDefaultPodLabels(&cntr)

		if cntr.Names[0] == "/"+containerName {
			container = &cntr
			break
		}
	}

	if container == nil {
		adapter.logger.Errorf("unable to find container for pod %s in namespace %s", podName, namespace)
		return nil, errors.ErrResourceNotFound
	}

	return container, nil
}

// getPodListFromContainers is responsible for retrieving a list of Kubernetes Pod objects
// based on the Docker containers running in a specific namespace. The function performs the following steps:
//
//  1. Prepares Docker container listing options. If the namespace is neither 'default' nor empty,
//     it adds a filter to only include containers that are part of the given Kubernetes namespace.
//
// 2. Calls the Docker API to list all containers that match the prepared listing options.
//
//  3. Invokes buildPodList to convert the list of Docker containers into a list of Kubernetes Pod objects.
//     During this conversion, each container's metadata and spec are translated to the corresponding fields in a Pod object.
//
// 4. Returns a PodList object, which is a collection of the generated Pod objects, wrapped with metadata.
//
// Parameters:
// - ctx: The context within which the function should operate. This is used for timeouts and cancellations.
// - namespace: The Kubernetes namespace in which to look for Pods. An empty or 'default' namespace applies special handling.
//
// Returns:
// - core.PodList: A list of Kubernetes Pods encapsulated in a PodList object, along with Kubernetes metadata.
// - error: An error object which could contain various types of errors including API call failures, JSON unmarshalling errors, etc.
func (adapter *KubeDockerAdapter) getPodListFromContainers(ctx context.Context, namespace string) (core.PodList, error) {
	listOptions := types.ContainerListOptions{All: true}
	if !isDefaultOrEmptyNamespace(namespace) {
		listOptions.Filters = filters.ByNamespace(namespace)
	}

	containers, err := adapter.cli.ContainerList(ctx, listOptions)
	if err != nil {
		return core.PodList{}, err
	}

	pods, err := adapter.buildPodList(containers, namespace)
	if err != nil {
		return core.PodList{}, err
	}

	return core.PodList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "PodList",
			APIVersion: "v1",
		},
		Items: pods,
	}, nil
}

// buildPodList is responsible for creating a list of Kubernetes Pod objects based on a given list of Docker containers.
// The function operates by filtering and converting Docker containers to Pods.
// If the specified namespace is neither 'default' nor empty, it will decorate existing containers with the namespace and workload labels if they are missing.
// This will allow support for containers that were created outside of k2d.
// Parameters:
//   - containers: A list of Docker containers from which the Pods will be created.
//   - namespace: The Kubernetes namespace to which the list of Pods should be restricted.
//     If the namespace is empty or 'default', special label handling will be applied.
//
// Returns:
//   - []core.Pod: A list of Kubernetes Pods constructed from the filtered list of Docker containers.
//   - error: An error object that may contain information about any error occurring during the conversion process,
//     such as issues in invoking the Docker API or converting the container attributes to Pod fields.
func (adapter *KubeDockerAdapter) buildPodList(containers []types.Container, namespace string) ([]core.Pod, error) {
	var pods []core.Pod

	for _, container := range containers {
		if isDefaultOrEmptyNamespace(namespace) {
			updateDefaultPodLabels(&container)
		}

		if !isContainerInNamespace(&container, namespace) {
			continue
		}

		pod, err := adapter.buildPodFromContainer(container)
		if err != nil {
			return nil, fmt.Errorf("unable to get pods: %w", err)
		}
		pods = append(pods, pod)
	}

	return pods, nil
}

// updateDefaultPodLabels is a utility function that sets the default pod labels associated to a Docker container
// if they are not already set. This is used for containers that were created outside of k2d.
// It sets the namespace label to 'default' if it is missing,
// and extracts the workload name from the Docker container's name.
//
// Parameters:
// - container: A pointer to the Docker container whose labels need to be updated.
func updateDefaultPodLabels(container *types.Container) {
	if _, exists := container.Labels[k2dtypes.NamespaceNameLabelKey]; !exists {
		container.Labels[k2dtypes.NamespaceNameLabelKey] = "default"
		container.Labels[k2dtypes.WorkloadNameLabelKey] = strings.TrimPrefix(container.Names[0], "/")
	}
}
