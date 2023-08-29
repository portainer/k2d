package filesystem

import (
	"bytes"
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/portainer/k2d/internal/adapter/store/errors"
	"github.com/portainer/k2d/pkg/filesystem"
	str "github.com/portainer/k2d/pkg/strings"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/kubernetes/pkg/apis/core"
)

func buildSecretMetadataFileName(secretName string) string {
	return fmt.Sprintf("%s-k2dsec.metadata", secretName)
}

func (s *FileSystemStore) DeleteSecret(secretName string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	files, err := os.ReadDir(s.secretPath)
	if err != nil {
		return fmt.Errorf("unable to read secret directory: %w", err)
	}

	fileNames := []string{}
	for _, file := range files {
		fileNames = append(fileNames, file.Name())
	}

	uniqueNames := str.UniquePrefixes(fileNames, SecretSeparator)

	if !str.IsStringInSlice(secretName, uniqueNames) {
		return fmt.Errorf("secret %s not found", secretName)
	}

	filePrefix := fmt.Sprintf("%s%s", secretName, SecretSeparator)

	for _, file := range files {
		if strings.HasPrefix(file.Name(), filePrefix) {
			err := os.Remove(path.Join(s.secretPath, file.Name()))
			if err != nil {
				return fmt.Errorf("unable to remove file %s: %w", file.Name(), err)
			}
		}
	}

	metadataFileName := buildSecretMetadataFileName(secretName)
	metadataFileFound, err := filesystem.FileExists(path.Join(s.secretPath, metadataFileName))
	if err != nil {
		return fmt.Errorf("unable to check if metadata file exists: %w", err)
	}

	if metadataFileFound {
		err := os.Remove(path.Join(s.secretPath, metadataFileName))
		if err != nil {
			return fmt.Errorf("unable to remove file %s: %w", metadataFileName, err)
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

func (s *FileSystemStore) GetSecret(secretName string) (*core.Secret, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	files, err := os.ReadDir(s.secretPath)
	if err != nil {
		return nil, fmt.Errorf("unable to read secret directory: %w", err)
	}

	fileNames := []string{}
	for _, file := range files {
		fileNames = append(fileNames, file.Name())
	}

	uniqueNames := str.UniquePrefixes(fileNames, SecretSeparator)

	if !str.IsStringInSlice(secretName, uniqueNames) {
		return nil, errors.ErrResourceNotFound
	}

	secret := core.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        secretName,
			Annotations: map[string]string{},
			Namespace:   "default",
		},
		Data: map[string][]byte{},
		Type: core.SecretTypeOpaque,
	}

	filePrefix := fmt.Sprintf("%s%s", secretName, SecretSeparator)

	for _, file := range files {
		if strings.HasPrefix(file.Name(), filePrefix) {
			data, err := os.ReadFile(path.Join(s.secretPath, file.Name()))
			if err != nil {
				return nil, fmt.Errorf("unable to read file %s: %w", file.Name(), err)
			}

			secret.Data[strings.TrimPrefix(file.Name(), secretName+SecretSeparator)] = bytes.TrimSuffix(data, []byte("\n"))
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

	metadataFileName := buildSecretMetadataFileName(secretName)
	metadataFileFound, err := filesystem.FileExists(path.Join(s.secretPath, metadataFileName))
	if err != nil {
		return nil, fmt.Errorf("unable to check if metadata file exists: %w", err)
	}

	if metadataFileFound {
		metadata, err := filesystem.LoadMetadataFromDisk(s.secretPath, metadataFileName)
		if err != nil {
			return nil, fmt.Errorf("unable to load secret metadata from disk: %w", err)
		}

		secret.Labels = metadata
	}

	return &secret, nil
}

func (s *FileSystemStore) GetSecrets(selector labels.Selector) (core.SecretList, error) {
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

	uniqueNames := str.UniquePrefixes(fileNames, SecretSeparator)
	uniqueNames = str.RemoveItemsWithSuffix(uniqueNames, ".metadata")

	secrets := []core.Secret{}

	for _, name := range uniqueNames {
		secret := core.Secret{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Secret",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: "default",
			},
			Data: map[string][]byte{},
			Type: core.SecretTypeOpaque,
		}

		metadataFileName := buildSecretMetadataFileName(secret.Name)
		metadataFileFound, err := filesystem.FileExists(path.Join(s.secretPath, metadataFileName))
		if err != nil {
			return core.SecretList{}, fmt.Errorf("unable to check if metadata file exists: %w", err)
		}

		if metadataFileFound {
			metadata, err := filesystem.LoadMetadataFromDisk(s.secretPath, metadataFileName)
			if err != nil {
				return core.SecretList{}, fmt.Errorf("unable to load secret metadata from disk: %w", err)
			}

			secret.Labels = metadata
		}

		if !selector.Matches(labels.Set(secret.Labels)) {
			continue
		}

		for _, file := range files {
			if strings.HasPrefix(file.Name(), fmt.Sprintf("%s%s", name, SecretSeparator)) {
				data, err := os.ReadFile(path.Join(s.secretPath, file.Name()))
				if err != nil {
					return core.SecretList{}, fmt.Errorf("unable to read file %s: %w", file.Name(), err)
				}

				secret.Data[strings.TrimPrefix(file.Name(), name+SecretSeparator)] = data
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

	data := map[string]string{}

	for key, value := range secret.Data {
		data[key] = string(value)
	}

	for key, value := range secret.StringData {
		data[key] = value
	}

	filePrefix := fmt.Sprintf("%s%s", secret.Name, SecretSeparator)
	err := filesystem.StoreDataMapOnDisk(s.secretPath, filePrefix, data)
	if err != nil {
		return err
	}

	if len(secret.Labels) != 0 {
		metadataFileName := buildSecretMetadataFileName(secret.Name)
		err = filesystem.StoreMetadataOnDisk(s.secretPath, metadataFileName, secret.Labels)
		if err != nil {
			return err
		}
	}

	return nil
}
