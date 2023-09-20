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
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/kubernetes/pkg/apis/core"
)

// DeleteSecret removes a secret identified by its name and namespace.
// The function performs the following tasks:
// 1. Locks the mutex to ensure thread-safety.
// 2. Reads the directory where secrets are stored.
// 3. Verifies if the secret file with the specified prefix exists.
// 4. If found, deletes the metadata file associated with the secret.
// 5. Iterates over all secret data files and removes them.
//
// Parameters:
//   - secretName: The name of the secret to be deleted.
//   - namespace: The namespace of the secret.
//
// Returns:
//   - error: Returns an error if any step of the deletion process fails,
//     otherwise returns nil.
func (s *FileSystemStore) DeleteSecret(secretName, namespace string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	metadataFileName := buildSecretMetadataFileName(secretName, namespace)
	metadataFilePath := path.Join(s.secretPath, metadataFileName)

	metadataFileExists, err := filesystem.FileExists(metadataFilePath)
	if err != nil {
		return fmt.Errorf("unable to check if secret metadata file %s exists: %w", metadataFileName, err)
	}

	if !metadataFileExists {
		return errors.ErrResourceNotFound
	}

	err = os.Remove(metadataFilePath)
	if err != nil {
		return fmt.Errorf("unable to remove secret metadata file %s: %w", metadataFileName, err)
	}

	files, err := os.ReadDir(s.secretPath)
	if err != nil {
		return fmt.Errorf("unable to read secret directory: %w", err)
	}

	filePrefix := buildSecretFilePrefix(secretName, namespace)
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
// Each bind is stored in a separate annotation and contains the filename of the file to mount inside the container and the path to the file on the host.
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

// GetSecret retrieves a specific secret identified by its name and namespace
// from a file system-based secret store. This function locks the secret store
// using a mutex to ensure thread-safety during the read operation.
//
// The function performs the following steps:
// 1. Reads all the files in the secret directory.
// 2. Searches for a specific secret file by its prefix which is a combination of its name and namespace.
// 3. Loads the metadata associated with the secret from the disk.
// 4. Creates a Secret object based on the loaded metadata.
// 5. Updates the Secret object with data loaded from the secret data file(s).
//
// Parameters:
// - secretName: The name of the secret to retrieve.
// - namespace: The namespace where the secret is located.
//
// Returns:
// - A pointer to the retrieved Secret object.
// - An error object if the function fails to retrieve the secret.
func (s *FileSystemStore) GetSecret(secretName, namespace string) (*core.Secret, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	metadataFileName := buildSecretMetadataFileName(secretName, namespace)
	metadataFilePath := path.Join(s.secretPath, metadataFileName)

	metadataFileExists, err := filesystem.FileExists(metadataFilePath)
	if err != nil {
		return nil, fmt.Errorf("unable to check if secret metadata file %s exists: %w", metadataFileName, err)
	}

	if !metadataFileExists {
		return nil, errors.ErrResourceNotFound
	}

	metadata, err := filesystem.LoadMetadataFromDisk(metadataFilePath)
	if err != nil {
		return nil, fmt.Errorf("unable to load secret metadata from disk: %w", err)
	}

	secret, err := createSecretFromMetadata(secretName, namespace, metadata)
	if err != nil {
		return nil, fmt.Errorf("unable to build secret from metadata: %w", err)
	}

	files, err := os.ReadDir(s.secretPath)
	if err != nil {
		return nil, fmt.Errorf("unable to read secret directory: %w", err)
	}

	filePrefix := buildSecretFilePrefix(secretName, namespace)
	for _, file := range files {
		if strings.HasPrefix(file.Name(), filePrefix) {

			err := s.updateSecretDataFromFile(&secret, file.Name())
			if err != nil {
				return nil, fmt.Errorf("unable to update secret data from file %s: %w", file.Name(), err)
			}
		}
	}

	return &secret, nil
}

// GetSecrets retrieves a list of secrets from a file system-based secret store
// that match the given namespace and selector labels. It locks the secret store
// with a mutex to ensure thread-safety during read operations.
//
// The function performs the following steps:
// 1. Reads all the files in the secret directory.
// 2. Segregates the files into metadata files and data files.
// 3. Builds a list of Secret objects based on the metadata files.
// 4. Filters the Secret objects based on the namespace and selector.
// 5. Updates the Secret objects with data loaded from the secret data files.
//
// Parameters:
// - namespace: The namespace where the secrets are located.
// - selector: Label selector to filter which secrets to retrieve.
//
// Returns:
// - A SecretList object containing all matching secrets.
// - An error object if the function fails to retrieve the secrets.
func (s *FileSystemStore) GetSecrets(namespace string, selector labels.Selector) (core.SecretList, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	files, err := os.ReadDir(s.secretPath)
	if err != nil {
		return core.SecretList{}, fmt.Errorf("unable to read secret directory: %w", err)
	}

	metadataFiles, dataFiles := s.isolateSecretMetadataAndDataFiles(files)

	secrets, err := s.buildSecrets(metadataFiles, dataFiles, namespace, selector)
	if err != nil {
		return core.SecretList{}, fmt.Errorf("unable to build secrets: %w", err)
	}

	return core.SecretList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "SecretList",
			APIVersion: "v1",
		},
		Items: secrets,
	}, nil
}

// StoreSecret stores a new secret or updates an existing one.
// The function performs the following tasks:
//  1. Locks the mutex to ensure thread-safety.
//  2. Prepares the labels for the secret, merging any existing labels.
//  3. Stores the metadata of the secret in the disk.
//  4. Iterates over the 'Data' and 'StringData' fields of the secret,
//     preparing the data to be stored.
//  5. Stores the prepared data on the disk.
//
// Parameters:
//   - secret: A pointer to the corev1.Secret object containing the secret data
//     to be stored.
//
// Returns:
//   - error: Returns an error if any step of the storage process fails,
//     otherwise returns nil.
func (s *FileSystemStore) StoreSecret(secret *corev1.Secret) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	labels := map[string]string{
		NamespaceNameLabelKey:     secret.Namespace,
		CreationTimestampLabelKey: time.Now().UTC().Format(time.RFC3339),
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

// isolateSecretMetadataAndDataFiles segregates the given directory entries into
// secret metadata files and data files based on their file name suffixes and prefixes.
func (s *FileSystemStore) isolateSecretMetadataAndDataFiles(files []os.DirEntry) ([]string, map[string][]string) {
	metadataFiles := []string{}
	dataFiles := map[string][]string{}

	for _, file := range files {
		// Skip non secret files
		if !strings.Contains(file.Name(), SecretSeparator) && !strings.Contains(file.Name(), "k2dsec.metadata") {
			continue
		}

		if strings.HasSuffix(file.Name(), ".metadata") {
			metadataFiles = append(metadataFiles, file.Name())
		} else {
			namespacedSecretName, _, err := getNamespacedSecretNameAndKeyFromFileName(file.Name())
			if err != nil {
				s.logger.Warnf("unable to get namespaced secret name from file name %s: %s", file.Name(), err.Error())
				continue
			}

			dataFiles[namespacedSecretName] = append(dataFiles[namespacedSecretName], file.Name())
		}
	}

	return metadataFiles, dataFiles
}

// buildSecrets constructs a list of Secret objects based on the given metadata
// and data files, namespace, and selector. It also updates the Secret objects
// with data loaded from the data files.
func (s *FileSystemStore) buildSecrets(metadataFiles []string, dataFiles map[string][]string, namespace string, selector labels.Selector) ([]core.Secret, error) {
	// Load metadata from disk and build initial secrets
	secrets, err := s.loadMetadataAndInitSecrets(metadataFiles, namespace, selector)
	if err != nil {
		return nil, err
	}

	// Populate secrets with data
	for namespacedSecretName, dataFiles := range dataFiles {
		for _, dataFile := range dataFiles {
			if secret, found := secrets[namespacedSecretName]; found {
				s.updateSecretDataFromFile(&secret, dataFile)
				secrets[namespacedSecretName] = secret
			}
		}
	}

	// Convert map values to slice
	secretsSlice := make([]core.Secret, 0, len(secrets))
	for _, secret := range secrets {
		secretsSlice = append(secretsSlice, secret)
	}

	return secretsSlice, nil
}

// loadMetadataAndInitSecrets loads secret metadata from disk and initializes a map
// of Secret objects based on the loaded metadata, namespace, and selector.
func (s *FileSystemStore) loadMetadataAndInitSecrets(metadataFiles []string, namespace string, selector labels.Selector) (map[string]core.Secret, error) {
	secrets := map[string]core.Secret{}

	for _, metadataFile := range metadataFiles {
		metadataFilePath := path.Join(s.secretPath, metadataFile)
		metadata, err := filesystem.LoadMetadataFromDisk(metadataFilePath)
		if err != nil {
			return secrets, fmt.Errorf("unable to load secret metadata from disk: %w", err)
		}

		if !selector.Matches(labels.Set(metadata)) {
			continue
		}

		namespaceName := metadata[NamespaceNameLabelKey]
		if namespace != "" && namespace != namespaceName {
			continue
		}

		namespacedSecretName := getNamespacedSecretNameFromMetadataFileName(metadataFile)
		secretName := getSecretNameFromNamespacedSecretName(namespacedSecretName, namespaceName)

		secret, err := createSecretFromMetadata(secretName, namespaceName, metadata)
		if err != nil {
			s.logger.Warnf("unable to build secret from metadata: %s", err.Error())
			continue
		}

		secrets[namespacedSecretName] = secret
	}

	return secrets, nil
}

// createSecretFromMetadata creates a new Secret object based on the given metadata,
// secret name, and namespace.
func createSecretFromMetadata(secretName, namespace string, metadata map[string]string) (core.Secret, error) {
	secret := core.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Labels:      metadata,
			Namespace:   namespace,
			Name:        secretName,
			Annotations: map[string]string{},
		},
		Data: map[string][]byte{},
		Type: core.SecretTypeOpaque,
	}

	creationTimestamp, ok := metadata[CreationTimestampLabelKey]
	if ok {
		parsedTime, err := time.Parse(time.RFC3339, creationTimestamp)
		if err != nil {
			return core.Secret{}, fmt.Errorf("unable to parse creation timestamp %s: %w", creationTimestamp, err)
		}
		secret.ObjectMeta.CreationTimestamp = metav1.NewTime(parsedTime)
	}

	return secret, nil
}

// updateSecretDataFromFile updates a Secret object with data loaded from a given
// data file.
func (s *FileSystemStore) updateSecretDataFromFile(secret *core.Secret, dataFile string) error {
	dataFilePath := path.Join(s.secretPath, dataFile)

	data, err := os.ReadFile(dataFilePath)
	if err != nil {
		return fmt.Errorf("unable to read secret data file %s: %w", dataFile, err)
	}

	_, secretKey, err := getNamespacedSecretNameAndKeyFromFileName(dataFile)
	if err != nil {
		return fmt.Errorf("unable to get secret key from file name %s: %w", dataFile, err)
	}

	secret.Data[secretKey] = data

	// The path to the file is stored in the annotation so that it can be mounted
	// inside a container by reading the store.k2d.io/filesystem/path/* annotations.
	// See the GetSecretBinds function for more details.
	secret.ObjectMeta.Annotations[fmt.Sprintf("%s/%s", FilePathAnnotationKey, secretKey)] = dataFilePath

	return nil
}
