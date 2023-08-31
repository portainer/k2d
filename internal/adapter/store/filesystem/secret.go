package filesystem

import (
	"bytes"
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/portainer/k2d/internal/adapter/errors"
	"github.com/portainer/k2d/pkg/filesystem"
	"github.com/portainer/k2d/pkg/maputils"
	str "github.com/portainer/k2d/pkg/strings"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/kubernetes/pkg/apis/core"
)

// TODO: add function comments

// Each secret has its own metadata file using the following naming convention:
// [namespace]-[secret-name]-k2dsec.metadata
func buildSecretMetadataFileName(secretName, namespace string) string {
	return fmt.Sprintf("%s-%s-k2dsec.metadata", namespace, secretName)
}

// Each key of a secret is stored in a separate file using the following naming convention:
// [namespace]-[secret-name]-k2dsec-[key]
func buildSecretFilePrefix(secretName, namespace string) string {
	return fmt.Sprintf("%s-%s%s", namespace, secretName, SecretSeparator)
}

func (s *FileSystemStore) DeleteSecret(secretName, namespace string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	files, err := os.ReadDir(s.secretPath)
	if err != nil {
		return fmt.Errorf("unable to read secret directory: %w", err)
	}

	filePrefix := buildSecretFilePrefix(secretName, namespace)

	// TODO: centralize this logic into a function hasMatchingSecretFile(files []os.FileInfo, filePrefix string) bool
	hasMatchingSecretFile := false
	for _, file := range files {
		if strings.HasPrefix(file.Name(), filePrefix) {
			hasMatchingSecretFile = true
			break
		}
	}

	if !hasMatchingSecretFile {
		return errors.ErrResourceNotFound
	}

	metadataFileName := buildSecretMetadataFileName(secretName, namespace)
	err = os.Remove(path.Join(s.secretPath, metadataFileName))
	if err != nil {
		return fmt.Errorf("unable to remove secret metadata file %s: %w", metadataFileName, err)
	}

	for _, file := range files {
		if strings.HasPrefix(file.Name(), filePrefix) {
			err := os.Remove(path.Join(s.secretPath, file.Name()))
			if err != nil {
				return fmt.Errorf("unable to remove secret data file %s: %w", file.Name(), err)
			}
		}
	}

	return nil
}

// The filesystem implementation will return a list of files that needs to be mounted
// for a specific Secret. This list is built from the store.k2d.io/filesystem/path/* annotations of the Secret.
// Each bind contains the filename of the file to mount inside the container and the path to the file on the host.
// The format of each bind is: filename:/path/to/matching/file
func (s *FileSystemStore) GetSecretBinds(secret *core.Secret) (map[string]string, error) {
	binds := map[string]string{}

	for key, value := range secret.Annotations {
		if strings.HasPrefix(key, FilePathAnnotationKey) {
			binds[strings.TrimPrefix(key, FilePathAnnotationKey+"/")] = value
		}
	}

	return binds, nil
}

func (s *FileSystemStore) GetSecret(secretName, namespace string) (*core.Secret, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	files, err := os.ReadDir(s.secretPath)
	if err != nil {
		return nil, fmt.Errorf("unable to read secret directory: %w", err)
	}

	filePrefix := buildSecretFilePrefix(secretName, namespace)

	// TODO: centralize this logic into a function hasMatchingSecretFile(files []os.FileInfo, filePrefix string) bool
	hasMatchingSecretFile := false
	for _, file := range files {
		if strings.HasPrefix(file.Name(), filePrefix) {
			hasMatchingSecretFile = true
			break
		}
	}

	if !hasMatchingSecretFile {
		return nil, errors.ErrResourceNotFound
	}

	secret := core.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        secretName,
			Namespace:   namespace,
			Annotations: map[string]string{},
		},
		Data: map[string][]byte{},
		Type: core.SecretTypeOpaque,
	}

	metadataFileName := buildSecretMetadataFileName(secretName, namespace)
	metadata, err := filesystem.LoadMetadataFromDisk(s.secretPath, metadataFileName)
	if err != nil {
		return nil, fmt.Errorf("unable to load secret metadata from disk: %w", err)
	}

	secret.Labels = metadata

	for _, file := range files {
		if strings.HasPrefix(file.Name(), filePrefix) {
			data, err := os.ReadFile(path.Join(s.secretPath, file.Name()))
			if err != nil {
				return nil, fmt.Errorf("unable to read file %s: %w", file.Name(), err)
			}

			secret.Data[strings.TrimPrefix(file.Name(), filePrefix)] = bytes.TrimSuffix(data, []byte("\n"))
			info, err := os.Stat(path.Join(s.secretPath, file.Name()))
			if err != nil {
				return nil, fmt.Errorf("unable to get file info for %s: %w", file.Name(), err)
			}

			secret.ObjectMeta.CreationTimestamp = metav1.NewTime(info.ModTime())

			// The path to the file is stored in the annotation so that it can be mounted
			// inside a container by reading the store.k2d.io/filesystem/path/* annotations.
			// See the GetSecretBinds function for more details.
			secret.ObjectMeta.Annotations[fmt.Sprintf("%s/%s", FilePathAnnotationKey, strings.TrimPrefix(file.Name(), filePrefix))] = path.Join(s.secretPath, file.Name())
		}
	}

	return &secret, nil
}

func (s *FileSystemStore) GetSecrets(namespace string, selector labels.Selector) (core.SecretList, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	files, err := os.ReadDir(s.secretPath)
	if err != nil {
		return core.SecretList{}, fmt.Errorf("unable to read secret directory: %w", err)
	}

	fileNames := []string{}
	for _, file := range files {
		fileNames = append(fileNames, file.Name())
	}

	// We first need to find all the unique secret names
	uniqueNames := str.RetrieveUniquePrefixes(fileNames, SecretSeparator)

	// We then need to filter out the secrets that are not in the namespace
	uniqueNames = str.FilterStringsByPrefix(uniqueNames, namespace)

	// We also need to filter out the metadata files
	uniqueNames = str.RemoveItemsWithSuffix(uniqueNames, ".metadata")

	secrets := []core.Secret{}
	for _, name := range uniqueNames {

		secret := core.Secret{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Secret",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{},
			Data:       map[string][]byte{},
			Type:       core.SecretTypeOpaque,
		}

		// TODO: find a better way to do this, this is dirty as it doesn't rely on the buildSecretMetadataFileName function
		metadataFileName := fmt.Sprintf("%s-k2dsec.metadata", name)
		metadata, err := filesystem.LoadMetadataFromDisk(s.secretPath, metadataFileName)
		if err != nil {
			return core.SecretList{}, fmt.Errorf("unable to load secret metadata from disk: %w", err)
		}

		secret.Labels = metadata
		secret.ObjectMeta.Namespace = metadata[NamespaceNameLabelKey]
		secret.ObjectMeta.Name = strings.TrimPrefix(name, secret.ObjectMeta.Namespace+"-")

		if !selector.Matches(labels.Set(secret.Labels)) {
			continue
		}

		filePrefix := buildSecretFilePrefix(secret.ObjectMeta.Name, secret.ObjectMeta.Namespace)
		for _, file := range files {
			if strings.HasPrefix(file.Name(), filePrefix) {
				data, err := os.ReadFile(path.Join(s.secretPath, file.Name()))
				if err != nil {
					return core.SecretList{}, fmt.Errorf("unable to read file %s: %w", file.Name(), err)
				}

				secret.Data[strings.TrimPrefix(file.Name(), filePrefix)] = data
				info, err := os.Stat(path.Join(s.secretPath, file.Name()))
				if err != nil {
					return core.SecretList{}, fmt.Errorf("unable to get file info for %s: %w", file.Name(), err)
				}
				secret.ObjectMeta.CreationTimestamp = metav1.NewTime(info.ModTime())
			}
		}

		secrets = append(secrets, secret)
	}

	return core.SecretList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "SecretList",
			APIVersion: "v1",
		},
		Items: secrets,
	}, nil
}

func (s *FileSystemStore) StoreSecret(secret *corev1.Secret) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	labels := map[string]string{
		NamespaceNameLabelKey: secret.Namespace,
	}
	maputils.MergeMapsInPlace(labels, secret.Labels)

	metadataFileName := buildSecretMetadataFileName(secret.Name, secret.Namespace)
	err := filesystem.StoreMetadataOnDisk(s.secretPath, metadataFileName, labels)
	if err != nil {
		return fmt.Errorf("unable to store secret metadata on disk: %w", err)
	}

	data := map[string]string{}

	for key, value := range secret.Data {
		data[key] = string(value)
	}

	for key, value := range secret.StringData {
		data[key] = value
	}

	filePrefix := buildSecretFilePrefix(secret.Name, secret.Namespace)
	err = filesystem.StoreDataMapOnDisk(s.secretPath, filePrefix, data)
	if err != nil {
		return err
	}

	return nil
}
