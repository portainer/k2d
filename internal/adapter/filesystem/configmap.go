package filesystem

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/portainer/k2d/pkg/filesystem"
	str "github.com/portainer/k2d/pkg/strings"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubernetes/pkg/apis/core"
)

var ErrConfigMapNotFound = errors.New("configmap file(s) not found")

// TODO: this package requires a lot of refactoring to make it more readable and maintainable

func (store *FileSystemStore) DeleteConfigMap(configMapName string) error {
	store.mutex.Lock()
	defer store.mutex.Unlock()

	files, err := os.ReadDir(store.configMapPath)
	if err != nil {
		return fmt.Errorf("unable to read configmap directory: %w", err)
	}

	fileNames := []string{}
	for _, file := range files {
		fileNames = append(fileNames, file.Name())
	}

	uniqueNames := str.UniquePrefixes(fileNames, CONFIGMAP_SEPARATOR)

	if !str.IsStringInSlice(configMapName, uniqueNames) {
		return fmt.Errorf("configmap %s not found", configMapName)
	}

	filePrefix := fmt.Sprintf("%s%s", configMapName, CONFIGMAP_SEPARATOR)

	for _, file := range files {
		if strings.HasPrefix(file.Name(), filePrefix) {
			err := os.Remove(fmt.Sprintf("%s/%s", store.configMapPath, file.Name()))
			if err != nil {
				return fmt.Errorf("unable to remove file %s: %w", file.Name(), err)
			}
		}
	}

	return nil
}

func (store *FileSystemStore) GetConfigMap(configMapName string) (*core.ConfigMap, error) {
	store.mutex.Lock()
	defer store.mutex.Unlock()

	files, err := os.ReadDir(store.configMapPath)
	if err != nil {
		return &core.ConfigMap{}, fmt.Errorf("unable to read configmap directory: %w", err)
	}

	fileNames := []string{}
	for _, file := range files {
		fileNames = append(fileNames, file.Name())
	}

	uniqueNames := str.UniquePrefixes(fileNames, CONFIGMAP_SEPARATOR)

	if !str.IsStringInSlice(configMapName, uniqueNames) {
		return &core.ConfigMap{}, ErrConfigMapNotFound
	}

	configMap := core.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        configMapName,
			Annotations: map[string]string{},
			Namespace:   "default",
		},
		Data: map[string]string{},
	}

	filePrefix := fmt.Sprintf("%s%s", configMapName, CONFIGMAP_SEPARATOR)

	for _, file := range files {
		if strings.HasPrefix(file.Name(), filePrefix) {
			data, err := os.ReadFile(fmt.Sprintf("%s/%s", store.configMapPath, file.Name()))
			if err != nil {
				return &core.ConfigMap{}, fmt.Errorf("unable to read file %s: %w", file.Name(), err)
			}

			configMap.Data[strings.TrimPrefix(file.Name(), configMapName+CONFIGMAP_SEPARATOR)] = strings.TrimSuffix(string(data), "\n")
			info, err := os.Stat(fmt.Sprintf("%s/%s", store.configMapPath, file.Name()))
			if err != nil {
				return &core.ConfigMap{}, fmt.Errorf("unable to get file info for %s: %w", file.Name(), err)
			}

			configMap.ObjectMeta.CreationTimestamp = metav1.NewTime(info.ModTime())
			configMap.ObjectMeta.Annotations[fmt.Sprintf("configmap.k2d.io/%s", file.Name())] = fmt.Sprintf("%s/%s", store.configMapPath, file.Name())
		}
	}

	return &configMap, nil
}

func (store *FileSystemStore) GetConfigMaps() (core.ConfigMapList, error) {
	store.mutex.Lock()
	defer store.mutex.Unlock()

	files, err := os.ReadDir(store.configMapPath)
	if err != nil {
		return core.ConfigMapList{}, fmt.Errorf("unable to read configmap directory: %w", err)
	}

	fileNames := []string{}
	for _, file := range files {
		fileNames = append(fileNames, file.Name())
	}

	uniqueNames := str.UniquePrefixes(fileNames, CONFIGMAP_SEPARATOR)

	configMaps := []core.ConfigMap{}
	for _, name := range uniqueNames {
		configMap := core.ConfigMap{
			TypeMeta: metav1.TypeMeta{
				Kind:       "ConfigMap",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: "default",
			},
			Data: map[string]string{},
		}

		for _, file := range files {
			if strings.HasPrefix(file.Name(), fmt.Sprintf("%s%s", name, CONFIGMAP_SEPARATOR)) {
				data, err := os.ReadFile(fmt.Sprintf("%s/%s", store.configMapPath, file.Name()))
				if err != nil {
					return core.ConfigMapList{}, fmt.Errorf("unable to read file %s: %w", file.Name(), err)
				}

				configMap.Data[strings.TrimPrefix(file.Name(), name+CONFIGMAP_SEPARATOR)] = string(data)
				info, err := os.Stat(fmt.Sprintf("%s/%s", store.configMapPath, file.Name()))
				if err != nil {
					return core.ConfigMapList{}, fmt.Errorf("unable to get file info for %s: %w", file.Name(), err)
				}
				configMap.ObjectMeta.CreationTimestamp = metav1.NewTime(info.ModTime())
			}
		}

		configMaps = append(configMaps, configMap)
	}

	return core.ConfigMapList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMapList",
			APIVersion: "v1",
		},
		Items: configMaps,
	}, nil
}

func (store *FileSystemStore) StoreConfigMap(configMap *corev1.ConfigMap) error {
	store.mutex.Lock()
	defer store.mutex.Unlock()

	filePrefix := fmt.Sprintf("%s%s", configMap.Name, CONFIGMAP_SEPARATOR)
	err := filesystem.StoreDataMapOnDisk(store.configMapPath, filePrefix, configMap.Data)
	if err != nil {
		return err
	}

	return nil
}