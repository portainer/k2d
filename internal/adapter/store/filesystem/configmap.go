package filesystem

import (
	"fmt"
	"os"
	"path"
	"strings"
	"time"

	"github.com/portainer/k2d/internal/adapter/errors"
	"github.com/portainer/k2d/pkg/filesystem"
	"github.com/portainer/k2d/pkg/maputils"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubernetes/pkg/apis/core"
)

// DeleteConfigMap deletes a specific ConfigMap identified by its name and namespace
// from a file system-based ConfigMap store. This function locks the ConfigMap store
// using a mutex to ensure thread-safety during the delete operation.
//
// The function performs the following steps:
// 1. Reads all the files in the ConfigMap directory.
// 2. Searches for a specific ConfigMap file by its prefix, which is a combination of its name and namespace.
// 3. Removes the metadata file associated with the ConfigMap from the disk.
// 4. Removes all data files associated with the ConfigMap from the disk.
//
// Parameters:
// - configMapName: The name of the ConfigMap to delete.
// - namespace: The namespace where the ConfigMap is located.
//
// Returns:
// - An error object if the function fails to delete the ConfigMap.
func (s *FileSystemStore) DeleteConfigMap(configMapName, namespace string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	metadataFileName := buildConfigMapMetadataFileName(configMapName, namespace)
	metadataFilePath := path.Join(s.configMapPath, metadataFileName)

	metadataFileExists, err := filesystem.FileExists(metadataFilePath)
	if err != nil {
		return fmt.Errorf("unable to check if configmap metadata file %s exists: %w", metadataFileName, err)
	}

	if !metadataFileExists {
		return errors.ErrResourceNotFound
	}

	err = os.Remove(metadataFilePath)
	if err != nil {
		return fmt.Errorf("unable to remove configmap metadata file %s: %w", metadataFileName, err)
	}

	files, err := os.ReadDir(s.configMapPath)
	if err != nil {
		return fmt.Errorf("unable to read configmap directory: %w", err)
	}

	filePrefix := buildConfigMapFilePrefix(configMapName, namespace)
	for _, file := range files {
		if strings.HasPrefix(file.Name(), filePrefix) {
			err := os.Remove(path.Join(s.configMapPath, file.Name()))
			if err != nil {
				return fmt.Errorf("unable to remove configmap data file %s: %w", file.Name(), err)
			}
		}
	}

	return nil
}

// The filesystem implementation will return a list of files that needs to be mounted
// for a specific ConfigMap. This list is built from the store.k2d.io/filesystem/path/* annotations of the ConfigMap.
// Each bind is stored in a separate annotation and contains the filename of the file to mount inside the container and the path to the file on the host.
// The format of each bind is: filename:/path/to/matching/file
func (s *FileSystemStore) GetConfigMapBinds(configMap *core.ConfigMap) (map[string]string, error) {
	binds := map[string]string{}

	for key, value := range configMap.Annotations {
		if strings.HasPrefix(key, FilePathAnnotationKey) {
			binds[strings.TrimPrefix(key, FilePathAnnotationKey+"/")] = value
		}
	}

	return binds, nil
}

// GetConfigMap retrieves a specific ConfigMap identified by its name and namespace
// from a file system-based ConfigMap store. This function locks the ConfigMap store
// using a mutex to ensure thread-safety during the read operation.
//
// The function performs the following steps:
// 1. Reads all the files in the ConfigMap directory.
// 2. Searches for a specific ConfigMap file by its prefix, which is a combination of its name and namespace.
// 3. Loads the metadata associated with the ConfigMap from the disk.
// 4. Creates a ConfigMap object based on the loaded metadata.
// 5. Updates the ConfigMap object with data loaded from the ConfigMap data file(s).
//
// Parameters:
// - configMapName: The name of the ConfigMap to retrieve.
// - namespace: The namespace where the ConfigMap is located.
//
// Returns:
// - A pointer to the retrieved ConfigMap object.
// - An error object if the function fails to retrieve the ConfigMap.
func (s *FileSystemStore) GetConfigMap(configMapName, namespace string) (*core.ConfigMap, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	metadataFileName := buildConfigMapMetadataFileName(configMapName, namespace)
	metadataFilePath := path.Join(s.configMapPath, metadataFileName)

	metadataFileExists, err := filesystem.FileExists(metadataFilePath)
	if err != nil {
		return nil, fmt.Errorf("unable to check if configmap metadata file %s exists: %w", metadataFileName, err)
	}

	if !metadataFileExists {
		return nil, errors.ErrResourceNotFound
	}

	metadata, err := filesystem.LoadMetadataFromDisk(metadataFilePath)
	if err != nil {
		return nil, fmt.Errorf("unable to load configmap metadata from disk: %w", err)
	}

	configMap, err := createConfigMapFromMetadata(configMapName, namespace, metadata)
	if err != nil {
		return nil, fmt.Errorf("unable to build configmap from metadata: %w", err)
	}

	files, err := os.ReadDir(s.configMapPath)
	if err != nil {
		return nil, fmt.Errorf("unable to read configmap directory: %w", err)
	}

	filePrefix := buildConfigMapFilePrefix(configMapName, namespace)
	for _, file := range files {
		if strings.HasPrefix(file.Name(), filePrefix) {
			err := s.updateConfigMapDataFromFile(&configMap, file.Name())
			if err != nil {
				return nil, fmt.Errorf("unable to update configmap data from file %s: %w", file.Name(), err)
			}
		}
	}

	return &configMap, nil
}

// GetConfigMaps retrieves all ConfigMaps for a given namespace from a
// file system-based ConfigMap store. This function locks the ConfigMap store
// using a mutex to ensure thread-safety during the read operation.
//
// The function performs the following steps:
// 1. Reads all the files in the ConfigMap directory.
// 2. Segregates the files into metadata and data files.
// 3. Builds ConfigMap objects based on the segregated files.
// 4. Returns a ConfigMapList containing all the constructed ConfigMaps.
//
// Parameters:
// - namespace: The namespace for which to retrieve ConfigMaps.
//
// Returns:
// - A ConfigMapList object containing all the ConfigMaps for the given namespace.
// - An error object if the function fails to retrieve the ConfigMaps.
func (s *FileSystemStore) GetConfigMaps(namespace string) (core.ConfigMapList, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	files, err := os.ReadDir(s.configMapPath)
	if err != nil {
		return core.ConfigMapList{}, fmt.Errorf("unable to read configmap directory: %w", err)
	}

	metadataFiles, dataFiles := s.isolateConfigMapMetadataAndDataFiles(files)

	configMaps, err := s.buildConfigMaps(metadataFiles, dataFiles, namespace)
	if err != nil {
		return core.ConfigMapList{}, fmt.Errorf("unable to build configmaps: %w", err)
	}

	return core.ConfigMapList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMapList",
			APIVersion: "v1",
		},
		Items: configMaps,
	}, nil
}

// StoreConfigMap stores a given ConfigMap object in a file system-based ConfigMap store.
// This function locks the ConfigMap store using a mutex to ensure thread-safety during the write operation.
//
// The function performs the following steps:
// 1. Merges any existing labels with new ones including namespace and creation timestamp.
// 2. Stores metadata associated with the ConfigMap on the disk.
// 3. Stores the ConfigMap data on the disk.
//
// Parameters:
// - configMap: A pointer to the ConfigMap object to store.
//
// Returns:
// - An error object if the function fails to store the ConfigMap.
func (s *FileSystemStore) StoreConfigMap(configMap *corev1.ConfigMap) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	labels := map[string]string{
		NamespaceNameLabelKey:     configMap.Namespace,
		CreationTimestampLabelKey: time.Now().UTC().Format(time.RFC3339),
	}
	maputils.MergeMapsInPlace(labels, configMap.Labels)

	metadataFileName := buildConfigMapMetadataFileName(configMap.Name, configMap.Namespace)
	err := filesystem.StoreMetadataOnDisk(s.configMapPath, metadataFileName, labels)
	if err != nil {
		return fmt.Errorf("unable to store configmap metadata on disk: %w", err)
	}

	filePrefix := buildConfigMapFilePrefix(configMap.Name, configMap.Namespace)
	err = filesystem.StoreDataMapOnDisk(s.configMapPath, filePrefix, configMap.Data)
	if err != nil {
		return fmt.Errorf("unable to store configmap data on disk: %w", err)
	}

	return nil
}

// isolateConfigMapMetadataAndDataFiles segregates the given directory entries into
// configmap metadata files and data files based on their file name suffixes and prefixes.
func (s *FileSystemStore) isolateConfigMapMetadataAndDataFiles(files []os.DirEntry) ([]string, map[string][]string) {
	metadataFiles := []string{}
	dataFiles := map[string][]string{}

	for _, file := range files {
		// Skip non configmap files
		if !strings.Contains(file.Name(), ConfigMapSeparator) && !strings.Contains(file.Name(), "k2dcm.metadata") {
			continue
		}

		if strings.HasSuffix(file.Name(), ".metadata") {
			metadataFiles = append(metadataFiles, file.Name())
		} else {
			namespacedConfigMapName, _, err := getNamespacedConfigMapNameAndKeyFromFileName(file.Name())
			if err != nil {
				s.logger.Warnf("unable to get namespaced configmap name from file name %s: %s", file.Name(), err.Error())
				continue
			}

			dataFiles[namespacedConfigMapName] = append(dataFiles[namespacedConfigMapName], file.Name())
		}
	}

	return metadataFiles, dataFiles
}

// buildConfigMaps constructs a list of ConfigMap objects based on the given metadata
// files, data files and namespace. It also updates the ConfigMap objects
// with data loaded from the data files.
func (s *FileSystemStore) buildConfigMaps(metadataFiles []string, dataFiles map[string][]string, namespace string) ([]core.ConfigMap, error) {
	// Load metadata from disk and build initial configmaps
	configMaps, err := s.loadMetadataAndInitConfigMaps(metadataFiles, namespace)
	if err != nil {
		return nil, err
	}

	// Populate configmaps with data
	for namespacedConfigMapName, dataFiles := range dataFiles {
		for _, dataFile := range dataFiles {
			if configMap, found := configMaps[namespacedConfigMapName]; found {
				s.updateConfigMapDataFromFile(&configMap, dataFile)
				configMaps[namespacedConfigMapName] = configMap
			}
		}
	}

	// Convert map values to slice
	configMapSlice := make([]core.ConfigMap, 0, len(configMaps))
	for _, configMap := range configMaps {
		configMapSlice = append(configMapSlice, configMap)
	}

	return configMapSlice, nil
}

// loadMetadataAndInitConfigMaps loads configmap metadata from disk and initializes a map
// of ConfigMap objects based on the loaded metadata and namespace.
func (s *FileSystemStore) loadMetadataAndInitConfigMaps(metadataFiles []string, namespace string) (map[string]core.ConfigMap, error) {
	configMaps := map[string]core.ConfigMap{}

	for _, metadataFile := range metadataFiles {
		metadataFilePath := path.Join(s.configMapPath, metadataFile)
		metadata, err := filesystem.LoadMetadataFromDisk(metadataFilePath)
		if err != nil {
			return configMaps, fmt.Errorf("unable to load configmap metadata from disk: %w", err)
		}

		namespaceName := metadata[NamespaceNameLabelKey]
		if namespace != "" && namespace != namespaceName {
			continue
		}

		namespacedConfigMapName := getNamespacedConfigMapNameFromMetadataFileName(metadataFile)
		configMapName := getConfigMapNameFromNamespacedConfigMapName(namespacedConfigMapName, namespaceName)

		configMap, err := createConfigMapFromMetadata(configMapName, namespaceName, metadata)
		if err != nil {
			s.logger.Warnf("unable to build configmap from metadata: %s", err.Error())
			continue
		}

		configMaps[namespacedConfigMapName] = configMap
	}

	return configMaps, nil
}

// createConfigMapFromMetadata creates a new ConfigMap object based on the given metadata,
// configmap name, and namespace.
func createConfigMapFromMetadata(configMapName, namespace string, metadata map[string]string) (core.ConfigMap, error) {
	configMap := core.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Labels:      metadata,
			Namespace:   namespace,
			Name:        configMapName,
			Annotations: map[string]string{},
		},
		Data: map[string]string{},
	}

	creationTimestamp, ok := metadata[CreationTimestampLabelKey]
	if ok {
		parsedTime, err := time.Parse(time.RFC3339, creationTimestamp)
		if err != nil {
			return core.ConfigMap{}, fmt.Errorf("unable to parse creation timestamp %s: %w", creationTimestamp, err)
		}
		configMap.ObjectMeta.CreationTimestamp = metav1.NewTime(parsedTime)
	}

	return configMap, nil
}

// updateConfigMapDataFromFile updates a ConfigMap object with data loaded from a given
// data file.
func (s *FileSystemStore) updateConfigMapDataFromFile(configMap *core.ConfigMap, dataFile string) error {
	dataFilePath := path.Join(s.configMapPath, dataFile)

	data, err := os.ReadFile(dataFilePath)
	if err != nil {
		return fmt.Errorf("unable to read configmap data file %s: %w", dataFile, err)
	}

	_, configMapKey, err := getNamespacedConfigMapNameAndKeyFromFileName(dataFile)
	if err != nil {
		return fmt.Errorf("unable to get configmap key from file name %s: %w", dataFile, err)
	}

	configMap.Data[configMapKey] = string(data)

	// The path to the file is stored in the annotation so that it can be mounted
	// inside a container by reading the store.k2d.io/filesystem/path/* annotations.
	// See the GetConfigMapBinds function for more details.
	configMap.ObjectMeta.Annotations[fmt.Sprintf("%s/%s", FilePathAnnotationKey, configMapKey)] = dataFilePath

	return nil
}
