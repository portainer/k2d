package volume

import (
	"context"
	"fmt"
	"time"

	"github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/errdefs"
	"github.com/portainer/k2d/internal/adapter/errors"
	"github.com/portainer/k2d/pkg/maputils"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/kubernetes/pkg/apis/core"
)

// DeleteSecret deletes a specific secret identified by its name and namespace
// from a Docker volume-based secret store.
//
// The function performs the following steps:
// 1. Builds the Docker volume name for the secret based on its name and namespace.
// 2. Removes the Docker volume associated with the secret.
//
// Parameters:
// - secretName: The name of the secret to delete.
// - namespace: The namespace where the secret is located.
//
// Returns:
// - An error object if the function fails to delete the secret.
func (s *VolumeStore) DeleteSecret(secretName, namespace string) error {
	volumeName := buildSecretVolumeName(secretName, namespace)

	err := s.cli.VolumeRemove(context.TODO(), volumeName, true)
	if err != nil {
		return fmt.Errorf("unable to remove Docker volume: %w", err)
	}

	return nil
}

// GetSecretBinds returns the volume names that need to be mounted for a given Secret.
// The volume name is derived from the labels on the Secret.
// It returns a single bind with an empty key (representing the file name in the container) and the volume name as value.
func (s *VolumeStore) GetSecretBinds(secret *core.Secret) (map[string]string, error) {
	return map[string]string{
		"": secret.Labels[VolumeNameLabelKey],
	}, nil
}

// GetSecret retrieves a specific secret identified by its name and namespace
// from a Docker volume-based secret store.
//
// The function performs the following steps:
// 1. Builds the Docker volume name for the secret based on its name and namespace.
// 2. Inspects the Docker volume to retrieve its details.
// 3. Creates a Secret object from the inspected Docker volume.
// 4. Fetches the data map from the Docker volume and associates it with the Secret object.
//
// Parameters:
// - secretName: The name of the secret to retrieve.
// - namespace: The namespace where the secret is located.
//
// Returns:
// - A pointer to the retrieved Secret object.
// - An error object if the function fails to retrieve the secret.
func (s *VolumeStore) GetSecret(secretName, namespace string) (*core.Secret, error) {
	volumeName := buildSecretVolumeName(secretName, namespace)

	volume, err := s.cli.VolumeInspect(context.TODO(), volumeName)
	if err != nil {
		if errdefs.IsNotFound(err) {
			return nil, errors.ErrResourceNotFound
		}
		return nil, fmt.Errorf("unable to inspect Docker volume: %w", err)
	}

	secret, err := createSecretFromVolume(&volume)
	if err != nil {
		return nil, fmt.Errorf("unable to build secret from volume: %w", err)
	}

	data, err := s.getDataMapFromVolume(volume.Name)
	if err != nil {
		return nil, fmt.Errorf("unable to get data map from volume: %w", err)
	}

	secret.Data = maputils.ConvertMapStringToStringSliceByte(data)

	return &secret, nil
}

// GetSecrets retrieves all secrets for a given namespace from a Docker volume-based secret store,
// optionally filtered by a set of labels.
//
// The function performs the following steps:
// 1. Creates a filter to list Docker volumes associated with secrets in the given namespace.
// 2. Lists the Docker volumes using the created filter.
// 3. Filters volumes based on label selectors.
// 4. Creates Secret objects from the filtered Docker volumes.
// 5. Fetches data maps from the Docker volumes and associates them with the Secret objects.
//
// Parameters:
// - namespace: The namespace for which to retrieve secrets.
// - selector: Label selector to filter secrets.
//
// Returns:
// - A SecretList object containing all the filtered secrets for the given namespace.
// - An error object if the function fails to retrieve the secrets.
func (s *VolumeStore) GetSecrets(namespace string, selector labels.Selector) (core.SecretList, error) {
	filter := secretListFilter(namespace)
	volumes, err := s.cli.VolumeList(context.TODO(), volume.ListOptions{
		Filters: filter,
	})
	if err != nil {
		return core.SecretList{}, fmt.Errorf("unable to list Docker volumes: %w", err)
	}

	filteredVolumes := []volume.Volume{}
	for _, volume := range volumes.Volumes {
		if selector.Matches(labels.Set(volume.Labels)) {
			filteredVolumes = append(filteredVolumes, *volume)
		}
	}

	secrets := core.SecretList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "SecretList",
			APIVersion: "v1",
		},
		Items: []core.Secret{},
	}

	volumeNames := []string{}
	for _, volume := range filteredVolumes {
		volumeNames = append(volumeNames, volume.Name)
	}

	volumeData, err := s.getDataMapsFromVolumes(volumeNames)
	if err != nil {
		return core.SecretList{}, fmt.Errorf("unable to get data maps from volumes: %w", err)
	}

	for _, volume := range filteredVolumes {
		secret, err := createSecretFromVolume(&volume)
		if err != nil {
			s.logger.Warnf("unable to build secret from volume %s: %w", volume.Name, err)
			continue
		}

		secret.Data = maputils.ConvertMapStringToStringSliceByte(volumeData[volume.Name])

		secrets.Items = append(secrets.Items, secret)
	}

	return secrets, nil
}

// StoreSecret stores a given Secret object in a Docker volume-based secret store.
//
// The function performs the following steps:
// 1. Builds the Docker volume name for the secret based on its name and namespace.
// 2. Creates a new Docker volume with the constructed name and attaches labels to it.
// 3. Copies both the data map and string data of the Secret to the created Docker volume.
//
// Parameters:
// - secret: A pointer to the Secret object to store.
//
// Returns:
// - An error object if the function fails to store the secret.
func (s *VolumeStore) StoreSecret(secret *corev1.Secret) error {
	volumeName := buildSecretVolumeName(secret.Name, secret.Namespace)

	labels := map[string]string{
		ResourceTypeLabelKey:  SecretResourceType,
		NamespaceNameLabelKey: secret.Namespace,
	}
	maputils.MergeMapsInPlace(labels, secret.Labels)

	volume, err := s.cli.VolumeCreate(context.TODO(), volume.CreateOptions{
		Name:   volumeName,
		Labels: labels,
	})
	if err != nil {
		return fmt.Errorf("unable to create Docker volume: %w", err)
	}

	data := map[string]string{}

	for key, value := range secret.Data {
		data[key] = string(value)
	}

	for key, value := range secret.StringData {
		data[key] = value
	}

	err = s.copyDataMapToVolume(volume.Name, data)
	if err != nil {
		return fmt.Errorf("unable to copy data map to volume: %w", err)
	}

	return nil
}

// createSecretFromVolume constructs a Kubernetes Secret object from a Docker volume.
// Returns a Secret object, and an error if any occurs (e.g., if the volume's creation timestamp is not parseable).
func createSecretFromVolume(volume *volume.Volume) (core.Secret, error) {
	namespace := volume.Labels[NamespaceNameLabelKey]

	secret := core.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        getSecretNameFromVolumeName(volume.Name, namespace),
			Annotations: map[string]string{},
			Namespace:   namespace,
			Labels:      volume.Labels,
		},
		Data: map[string][]byte{},
		Type: core.SecretTypeOpaque,
	}

	secret.Labels[VolumeNameLabelKey] = volume.Name

	parsedTime, err := time.Parse(time.RFC3339, volume.CreatedAt)
	if err != nil {
		return core.Secret{}, err
	}

	secret.ObjectMeta.CreationTimestamp = metav1.NewTime(parsedTime)

	return secret, nil
}
