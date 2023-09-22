package volume

import (
	"context"
	"fmt"
	"time"

	"github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/errdefs"
	"github.com/portainer/k2d/internal/adapter/errors"
	"github.com/portainer/k2d/internal/adapter/types"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubernetes/pkg/apis/core"
)

// DeleteConfigMap deletes a specific ConfigMap identified by its name and namespace
// from a Docker volume-based ConfigMap store.
//
// The function performs the following steps:
// 1. Builds the Docker volume name for the ConfigMap based on its name and namespace.
// 2. Removes the Docker volume associated with the ConfigMap.
//
// Parameters:
// - configMapName: The name of the ConfigMap to delete.
// - namespace: The namespace where the ConfigMap is located.
//
// Returns:
// - An error object if the function fails to delete the ConfigMap.
func (store *VolumeStore) DeleteConfigMap(configMapName, namespace string) error {
	volumeName := buildConfigMapVolumeName(configMapName, namespace)

	err := store.cli.VolumeRemove(context.TODO(), volumeName, true)
	if err != nil {
		return fmt.Errorf("unable to remove Docker volume: %w", err)
	}

	return nil
}

// GetConfigMapBinds returns the volume names that need to be mounted for a given ConfigMap.
// The volume name is derived from the labels on the ConfigMap.
// It returns a single bind with an empty key (representing the file name in the container) and the volume name as value.
func (s *VolumeStore) GetConfigMapBinds(configMap *core.ConfigMap) (map[string]string, error) {
	return map[string]string{
		"": configMap.Labels[VolumeNameLabelKey],
	}, nil
}

// GetConfigMap retrieves a specific ConfigMap identified by its name and namespace
// from a Docker volume-based ConfigMap store.
//
// The function performs the following steps:
// 1. Builds the Docker volume name for the ConfigMap based on its name and namespace.
// 2. Inspects the Docker volume to retrieve its details.
// 3. Creates a ConfigMap object from the inspected Docker volume.
// 4. Fetches the data map from the Docker volume and associates it with the ConfigMap object.
//
// Parameters:
// - configMapName: The name of the ConfigMap to retrieve.
// - namespace: The namespace where the ConfigMap is located.
//
// Returns:
// - A pointer to the retrieved ConfigMap object.
// - An error object if the function fails to retrieve the ConfigMap.
func (s *VolumeStore) GetConfigMap(configMapName, namespace string) (*core.ConfigMap, error) {
	volumeName := buildConfigMapVolumeName(configMapName, namespace)

	volume, err := s.cli.VolumeInspect(context.TODO(), volumeName)
	if err != nil {
		if errdefs.IsNotFound(err) {
			return nil, errors.ErrResourceNotFound
		}
		return nil, fmt.Errorf("unable to inspect Docker volume: %w", err)
	}

	configMap, err := createConfigMapFromVolume(&volume)
	if err != nil {
		return nil, fmt.Errorf("unable to build config map from volume: %w", err)
	}

	data, err := s.getDataMapFromVolume(volume.Name)
	if err != nil {
		return nil, fmt.Errorf("unable to get data map from volume: %w", err)
	}

	configMap.Data = data

	return &configMap, nil
}

// GetConfigMaps retrieves all ConfigMaps for a given namespace from a
// Docker volume-based ConfigMap store.
//
// The function performs the following steps:
// 1. Creates a filter to list Docker volumes associated with ConfigMaps in the given namespace.
// 2. Lists the Docker volumes using the created filter.
// 3. Creates ConfigMap objects from the listed Docker volumes.
// 4. Fetches data maps from the Docker volumes and associates them with the ConfigMap objects.
//
// Parameters:
// - namespace: The namespace for which to retrieve ConfigMaps.
//
// Returns:
// - A ConfigMapList object containing all the ConfigMaps for the given namespace.
// - An error object if the function fails to retrieve the ConfigMaps.
func (store *VolumeStore) GetConfigMaps(namespace string) (core.ConfigMapList, error) {
	filter := configMapListFilter(namespace)
	volumes, err := store.cli.VolumeList(context.TODO(), volume.ListOptions{
		Filters: filter,
	})
	if err != nil {
		return core.ConfigMapList{}, fmt.Errorf("unable to list Docker volumes: %w", err)
	}

	configMaps := core.ConfigMapList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMapList",
			APIVersion: "v1",
		},
		Items: []core.ConfigMap{},
	}

	volumeNames := []string{}
	for _, volume := range volumes.Volumes {
		volumeNames = append(volumeNames, volume.Name)
	}

	volumeData, err := store.getDataMapsFromVolumes(volumeNames)
	if err != nil {
		return core.ConfigMapList{}, fmt.Errorf("unable to get data maps from volumes: %w", err)
	}

	for _, volume := range volumes.Volumes {
		configMap, err := createConfigMapFromVolume(volume)
		if err != nil {
			store.logger.Warnf("unable to build config map from volume %s: %w", volume.Name, err)
			continue
		}

		configMap.Data = volumeData[volume.Name]

		configMaps.Items = append(configMaps.Items, configMap)
	}

	return configMaps, nil
}

// StoreConfigMap stores a given ConfigMap object in a Docker volume-based ConfigMap store.
//
// The function performs the following steps:
// 1. Builds the Docker volume name for the ConfigMap based on its name and namespace.
// 2. Creates a new Docker volume with the constructed name and attaches labels to it.
// 3. Copies the data map of the ConfigMap to the created Docker volume.
//
// Parameters:
// - configMap: A pointer to the ConfigMap object to store.
//
// Returns:
// - An error object if the function fails to store the ConfigMap.
func (store *VolumeStore) StoreConfigMap(configMap *corev1.ConfigMap) error {
	volumeName := buildConfigMapVolumeName(configMap.Name, configMap.Namespace)

	// override is required for the configmap to be treated as the k2d system configmap
	configMap.Labels[types.NamespaceNameLabelKey] = configMap.Namespace
	configMap.Labels[ResourceTypeLabelKey] = ConfigMapResourceType

	volume, err := store.cli.VolumeCreate(context.TODO(), volume.CreateOptions{
		Name:   volumeName,
		Labels: configMap.Labels,
	})
	if err != nil {
		return fmt.Errorf("unable to create Docker volume: %w", err)
	}

	err = store.copyDataMapToVolume(volume.Name, configMap.Data)
	if err != nil {
		return fmt.Errorf("unable to copy data map to volume: %w", err)
	}

	return nil
}

// createConfigMapFromVolume constructs a Kubernetes ConfigMap object from a Docker volume.
// Returns a ConfigMap object, and an error if any occurs (e.g., if the volume's creation timestamp is not parseable).
func createConfigMapFromVolume(volume *volume.Volume) (core.ConfigMap, error) {
	namespace := volume.Labels[types.NamespaceNameLabelKey]

	configMap := core.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        getConfigMapNameFromVolumeName(volume.Name, namespace),
			Annotations: map[string]string{},
			Namespace:   namespace,
			Labels:      volume.Labels,
		},
		Data: map[string]string{},
	}

	configMap.Labels[VolumeNameLabelKey] = volume.Name

	parsedTime, err := time.Parse(time.RFC3339, volume.CreatedAt)
	if err != nil {
		return core.ConfigMap{}, err
	}

	configMap.ObjectMeta.CreationTimestamp = metav1.NewTime(parsedTime)

	return configMap, nil
}
