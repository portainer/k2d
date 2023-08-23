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

func (store *FileSystemStore) GetSecret(secretName string) (*core.Secret, error) {
	store.mutex.Lock()
	defer store.mutex.Unlock()

	// this is required for kubectl apply -f that executes kubectl get first
	_, err := os.Stat(path.Join(store.path, secretName, "_data"))
	if os.IsNotExist(err) {
		err = filesystem.CreateDir(path.Join(store.path, secretName, "_data"))
		if err != nil {
			return nil, fmt.Errorf("unable to create directory %s: %w", store.path+"/"+secretName+"/_data/", err)
		}
	}

	files, err := os.ReadDir(path.Join(store.path, secretName, "_data"))
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
			data, err := os.ReadFile(path.Join(store.path, secretName, "_data", file.Name()))
			if err != nil {
				return &core.Secret{}, fmt.Errorf("unable to read file %s: %w", file.Name(), err)
			}

			secret.Data[strings.TrimPrefix(file.Name(), secretName+SECRET_SEPARATOR)] = bytes.TrimSuffix(data, []byte("\n"))
			info, err := os.Stat(path.Join(store.path, secretName, "_data", file.Name()))
			if err != nil {
				return &core.Secret{}, fmt.Errorf("unable to get file info for %s: %w", file.Name(), err)
			}
			secret.ObjectMeta.CreationTimestamp = metav1.NewTime(info.ModTime())
			secret.ObjectMeta.Annotations[fmt.Sprintf("secret.k2d.io/%s", file.Name())] = secretName
		}
	}

	return &secret, nil
}

func (store *FileSystemStore) GetSecrets(mountPoints []string) (core.SecretList, error) {
	store.mutex.Lock()
	defer store.mutex.Unlock()

	secrets := []core.Secret{}
	for _, mountPoint := range mountPoints {
		files, err := os.ReadDir(mountPoint)
		if err != nil {
			return core.SecretList{}, fmt.Errorf("unable to read secret directory: %w", err)
		}

		fileNames := []string{}
		for _, file := range files {
			fileNames = append(fileNames, file.Name())
		}

		uniqueNames := str.UniquePrefixes(fileNames, SECRET_SEPARATOR)
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
					data, err := os.ReadFile(path.Join(mountPoint, file.Name()))
					if err != nil {
						return core.SecretList{}, fmt.Errorf("unable to read file %s: %w", file.Name(), err)
					}

					secret.Data[strings.TrimPrefix(file.Name(), name+SECRET_SEPARATOR)] = data
					info, err := os.Stat(path.Join(mountPoint, file.Name()))
					if err != nil {
						return core.SecretList{}, fmt.Errorf("unable to get file info for %s: %w", file.Name(), err)
					}
					secret.ObjectMeta.CreationTimestamp = metav1.NewTime(info.ModTime())
				}
			}

			secrets = append(secrets, secret)
		}
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

	err := filesystem.CreateDir(path.Join(store.path, secret.Name, "_data"))
	if err != nil {
		return fmt.Errorf("unable to create directory %s: %w", store.path+"/"+secret.Name+"/_data/", err)
	}

	filePrefix := fmt.Sprintf("%s%s", secret.Name, SECRET_SEPARATOR)
	err = filesystem.StoreDataMapOnDisk(path.Join(store.path, secret.Name, "_data"), filePrefix, data)
	if err != nil {
		return err
	}

	return nil
}
