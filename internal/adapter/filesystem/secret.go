package filesystem

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/portainer/k2d/pkg/filesystem"
	str "github.com/portainer/k2d/pkg/strings"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubernetes/pkg/apis/core"
)

var ErrSecretNotFound = errors.New("secret file(s) not found")

func buildSecretMetadataFileName(secretName string) string {
	return fmt.Sprintf("%s-k2dsec.metadata", secretName)
}

func (store *FileSystemStore) DeleteSecret(secretName string) error {
	store.mutex.Lock()
	defer store.mutex.Unlock()

	files, err := os.ReadDir(store.secretPath)
	if err != nil {
		return fmt.Errorf("unable to read secret directory: %w", err)
	}

	fileNames := []string{}
	for _, file := range files {
		fileNames = append(fileNames, file.Name())
	}

	uniqueNames := str.UniquePrefixes(fileNames, SECRET_SEPARATOR)

	if !str.IsStringInSlice(secretName, uniqueNames) {
		return fmt.Errorf("secret %s not found", secretName)
	}

	filePrefix := fmt.Sprintf("%s%s", secretName, SECRET_SEPARATOR)

	for _, file := range files {
		if strings.HasPrefix(file.Name(), filePrefix) {
			err := os.Remove(path.Join(store.secretPath, file.Name()))
			if err != nil {
				return fmt.Errorf("unable to remove file %s: %w", file.Name(), err)
			}
		}
	}

	metadataFileName := buildSecretMetadataFileName(secretName)
	metadataFileFound, err := filesystem.FileExists(path.Join(store.secretPath, metadataFileName))
	if err != nil {
		return fmt.Errorf("unable to check if metadata file exists: %w", err)
	}

	if metadataFileFound {
		err := os.Remove(path.Join(store.secretPath, metadataFileName))
		if err != nil {
			return fmt.Errorf("unable to remove file %s: %w", metadataFileName, err)
		}
	}

	return nil
}

func (store *FileSystemStore) GetSecret(secretName string) (*core.Secret, error) {
	store.mutex.Lock()
	defer store.mutex.Unlock()

	files, err := os.ReadDir(store.secretPath)
	if err != nil {
		return &core.Secret{}, fmt.Errorf("unable to read secret directory: %w", err)
	}

	fileNames := []string{}
	for _, file := range files {
		fileNames = append(fileNames, file.Name())
	}

	uniqueNames := str.UniquePrefixes(fileNames, SECRET_SEPARATOR)

	if !str.IsStringInSlice(secretName, uniqueNames) {
		return &core.Secret{}, ErrSecretNotFound
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

	filePrefix := fmt.Sprintf("%s%s", secretName, SECRET_SEPARATOR)

	for _, file := range files {
		if strings.HasPrefix(file.Name(), filePrefix) {
			data, err := os.ReadFile(path.Join(store.secretPath, file.Name()))
			if err != nil {
				return &core.Secret{}, fmt.Errorf("unable to read file %s: %w", file.Name(), err)
			}

			secret.Data[strings.TrimPrefix(file.Name(), secretName+SECRET_SEPARATOR)] = bytes.TrimSuffix(data, []byte("\n"))
			info, err := os.Stat(path.Join(store.secretPath, file.Name()))
			if err != nil {
				return &core.Secret{}, fmt.Errorf("unable to get file info for %s: %w", file.Name(), err)
			}
			secret.ObjectMeta.CreationTimestamp = metav1.NewTime(info.ModTime())
			secret.ObjectMeta.Annotations[fmt.Sprintf("secret.k2d.io/%s", file.Name())] = path.Join(store.secretPath, file.Name())
		}
	}

	metadataFileName := buildSecretMetadataFileName(secretName)
	metadataFileFound, err := filesystem.FileExists(path.Join(store.secretPath, metadataFileName))
	if err != nil {
		return &core.Secret{}, fmt.Errorf("unable to check if metadata file exists: %w", err)
	}

	if metadataFileFound {
		metadata, err := filesystem.LoadMetadataFromDisk(store.secretPath, metadataFileName)
		if err != nil {
			return &core.Secret{}, fmt.Errorf("unable to load secret metadata from disk: %w", err)
		}

		secret.Labels = metadata
	}

	return &secret, nil
}

func (store *FileSystemStore) GetSecrets() (core.SecretList, error) {
	store.mutex.Lock()
	defer store.mutex.Unlock()

	files, err := os.ReadDir(store.secretPath)
	if err != nil {
		return core.SecretList{}, fmt.Errorf("unable to read secret directory: %w", err)
	}

	fileNames := []string{}
	for _, file := range files {
		fileNames = append(fileNames, file.Name())
	}

	uniqueNames := str.UniquePrefixes(fileNames, SECRET_SEPARATOR)
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

		for _, file := range files {
			if strings.HasPrefix(file.Name(), fmt.Sprintf("%s%s", name, SECRET_SEPARATOR)) {
				data, err := os.ReadFile(path.Join(store.secretPath, file.Name()))
				if err != nil {
					return core.SecretList{}, fmt.Errorf("unable to read file %s: %w", file.Name(), err)
				}

				secret.Data[strings.TrimPrefix(file.Name(), name+SECRET_SEPARATOR)] = data
				info, err := os.Stat(path.Join(store.secretPath, file.Name()))
				if err != nil {
					return core.SecretList{}, fmt.Errorf("unable to get file info for %s: %w", file.Name(), err)
				}
				secret.ObjectMeta.CreationTimestamp = metav1.NewTime(info.ModTime())
			}
		}

		metadataFileName := buildSecretMetadataFileName(secret.Name)
		metadataFileFound, err := filesystem.FileExists(path.Join(store.secretPath, metadataFileName))
		if err != nil {
			return core.SecretList{}, fmt.Errorf("unable to check if metadata file exists: %w", err)
		}

		if metadataFileFound {
			metadata, err := filesystem.LoadMetadataFromDisk(store.secretPath, metadataFileName)
			if err != nil {
				return core.SecretList{}, fmt.Errorf("unable to load secret metadata from disk: %w", err)
			}

			secret.Labels = metadata
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

func (store *FileSystemStore) StoreSecret(secret *corev1.Secret) error {
	store.mutex.Lock()
	defer store.mutex.Unlock()

	data := map[string]string{}

	for key, value := range secret.Data {
		data[key] = string(value)
	}

	for key, value := range secret.StringData {
		data[key] = value
	}

	filePrefix := fmt.Sprintf("%s%s", secret.Name, SECRET_SEPARATOR)
	err := filesystem.StoreDataMapOnDisk(store.secretPath, filePrefix, data)
	if err != nil {
		return err
	}

	if len(secret.Labels) != 0 {
		metadataFileName := buildSecretMetadataFileName(secret.Name)
		err = filesystem.StoreMetadataOnDisk(store.secretPath, metadataFileName, secret.Labels)
		if err != nil {
			return err
		}
	}

	return nil
}
