package filesystem

import (
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/portainer/k2d/internal/adapter/store/errors"
	"github.com/portainer/k2d/pkg/filesystem"
	str "github.com/portainer/k2d/pkg/strings"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubernetes/pkg/apis/core"
)

// TODO: this package requires a lot of refactoring to make it more readable and maintainable
// It shares a lot of commonalities with the secret.go file

func buildConfigMapMetadataFileName(configMapName string) string {
	return fmt.Sprintf("%s-k2dcm.metadata", configMapName)
}

func (s *FileSystemStore) DeleteConfigMap(configMapName string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	files, err := os.ReadDir(s.configMapPath)
	if err != nil {
		return fmt.Errorf("unable to read configmap directory: %w", err)
	}

	fileNames := []string{}
	for _, file := range files {
		fileNames = append(fileNames, file.Name())
	}

	uniqueNames := str.UniquePrefixes(fileNames, ConfigMapSeparator)

	if !str.IsStringInSlice(configMapName, uniqueNames) {
		return fmt.Errorf("configmap %s not found", configMapName)
	}

	filePrefix := fmt.Sprintf("%s%s", configMapName, ConfigMapSeparator)

	for _, file := range files {
		if strings.HasPrefix(file.Name(), filePrefix) {
			err := os.Remove(path.Join(s.configMapPath, file.Name()))
			if err != nil {
				return fmt.Errorf("unable to remove file %s: %w", file.Name(), err)
			}
		}
	}

	metadataFileName := buildConfigMapMetadataFileName(configMapName)
	metadataFileFound, err := filesystem.FileExists(path.Join(s.configMapPath, metadataFileName))
	if err != nil {
		return fmt.Errorf("unable to check if metadata file exists: %w", err)
	}

	if metadataFileFound {
		err := os.Remove(path.Join(s.configMapPath, metadataFileName))
		if err != nil {
			return fmt.Errorf("unable to remove file %s: %w", metadataFileName, err)
		}
	}

	return nil
}

// The filesystem implementation will return a list of files that needs to be mounted
// for a specific ConfigMap. This list is built from the store.k2d.io/filesystem/path/* annotations of the ConfigMap.
func (s *FileSystemStore) GetConfigMapBinds(configMap *core.ConfigMap) ([]string, error) {
	binds := []string{}

	for key, value := range configMap.Annotations {
		if strings.HasPrefix(key, FilePathAnnotationKey) {
			binds = append(binds, value)
		}
	}

	return binds, nil
}

func (s *FileSystemStore) GetConfigMap(configMapName string) (*core.ConfigMap, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	files, err := os.ReadDir(s.configMapPath)
	if err != nil {
		return nil, fmt.Errorf("unable to read configmap directory: %w", err)
	}

	fileNames := []string{}
	for _, file := range files {
		fileNames = append(fileNames, file.Name())
	}

	uniqueNames := str.UniquePrefixes(fileNames, ConfigMapSeparator)

	if !str.IsStringInSlice(configMapName, uniqueNames) {
		return nil, errors.ErrResourceNotFound
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

	filePrefix := fmt.Sprintf("%s%s", configMapName, ConfigMapSeparator)

	for _, file := range files {
		if strings.HasPrefix(file.Name(), filePrefix) {
			data, err := os.ReadFile(path.Join(s.configMapPath, file.Name()))
			if err != nil {
				return nil, fmt.Errorf("unable to read file %s: %w", file.Name(), err)
			}

			configMap.Data[strings.TrimPrefix(file.Name(), configMapName+ConfigMapSeparator)] = string(data)
			info, err := os.Stat(path.Join(s.configMapPath, file.Name()))
			if err != nil {
				return nil, fmt.Errorf("unable to get file info for %s: %w", file.Name(), err)
			}

			configMap.ObjectMeta.CreationTimestamp = metav1.NewTime(info.ModTime())

			// The path to the file is stored in the annotation so that it can be mounted
			// inside a container by reading the store.k2d.io/filesystem/path/* annotations.
			configMap.ObjectMeta.Annotations[fmt.Sprintf("%s/%s", FilePathAnnotationKey, file.Name())] = path.Join(s.configMapPath, file.Name())
		}
	}

	metadataFileName := buildConfigMapMetadataFileName(configMapName)
	metadataFileFound, err := filesystem.FileExists(path.Join(s.configMapPath, metadataFileName))
	if err != nil {
		return nil, fmt.Errorf("unable to check if metadata file exists: %w", err)
	}

	if metadataFileFound {
		metadata, err := filesystem.LoadMetadataFromDisk(s.configMapPath, metadataFileName)
		if err != nil {
			return nil, fmt.Errorf("unable to load configmap metadata from disk: %w", err)
		}

		configMap.Labels = metadata
	}

	return &configMap, nil
}

func (s *FileSystemStore) GetConfigMaps() (core.ConfigMapList, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	files, err := os.ReadDir(s.configMapPath)
	if err != nil {
		return core.ConfigMapList{}, fmt.Errorf("unable to read configmap directory: %w", err)
	}

	fileNames := []string{}
	for _, file := range files {
		fileNames = append(fileNames, file.Name())
	}

	uniqueNames := str.UniquePrefixes(fileNames, ConfigMapSeparator)
	uniqueNames = str.RemoveItemsWithSuffix(uniqueNames, ".metadata")

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
			if strings.HasPrefix(file.Name(), fmt.Sprintf("%s%s", name, ConfigMapSeparator)) {
				data, err := os.ReadFile(path.Join(s.configMapPath, file.Name()))
				if err != nil {
					return core.ConfigMapList{}, fmt.Errorf("unable to read file %s: %w", file.Name(), err)
				}

				configMap.Data[strings.TrimPrefix(file.Name(), name+ConfigMapSeparator)] = string(data)
				info, err := os.Stat(path.Join(s.configMapPath, file.Name()))
				if err != nil {
					return core.ConfigMapList{}, fmt.Errorf("unable to get file info for %s: %w", file.Name(), err)
				}
				configMap.ObjectMeta.CreationTimestamp = metav1.NewTime(info.ModTime())
			}
		}

		metadataFileName := buildConfigMapMetadataFileName(name)
		metadataFileFound, err := filesystem.FileExists(path.Join(s.configMapPath, metadataFileName))
		if err != nil {
			return core.ConfigMapList{}, fmt.Errorf("unable to check if metadata file exists: %w", err)
		}

		if metadataFileFound {
			metadata, err := filesystem.LoadMetadataFromDisk(s.configMapPath, metadataFileName)
			if err != nil {
				return core.ConfigMapList{}, fmt.Errorf("unable to load configmap metadata from disk: %w", err)
			}

			configMap.Labels = metadata
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

func (s *FileSystemStore) StoreConfigMap(configMap *corev1.ConfigMap) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	filePrefix := fmt.Sprintf("%s%s", configMap.Name, ConfigMapSeparator)
	err := filesystem.StoreDataMapOnDisk(s.configMapPath, filePrefix, configMap.Data)
	if err != nil {
		return err
	}

	if len(configMap.Labels) != 0 {
		metadataFileName := buildConfigMapMetadataFileName(configMap.Name)
		err = filesystem.StoreMetadataOnDisk(s.configMapPath, metadataFileName, configMap.Labels)
		if err != nil {
			return err
		}
	}

	return nil
}
