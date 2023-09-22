package filesystem

import (
	"fmt"
	"path"
	"sync"

	"github.com/portainer/k2d/pkg/filesystem"
	"go.uber.org/zap"
)

const (
	// ConfigMapFolder is the name of the directory where ConfigMaps are stored
	ConfigMapFolder = "configmaps"

	// ConfigMapSeparator is the separator that is used to build the name of a ConfigMap file
	ConfigMapSeparator = "-k2dcm-"

	// SecretFolder is the name of the directory where Secrets are stored
	SecretFolder = "secrets"

	// SecretSeparator is the separator that is used to build the name of a Secret file
	SecretSeparator = "-k2dsec-"
)

const (
	// CreationTimestampLabelKey is the key used to store the creation timestamp of a Configmap or Secret resource
	// in the associated metadata file
	CreationTimestampLabelKey = "store.k2d.io/filesystem/creation-timestamp"

	// FilePathAnnotationKey is the key used to store the path to a data file for a ConfigMap or Secret resource
	// It is used to construct binds when mounting these files in containers
	FilePathAnnotationKey = "store.k2d.io/filesystem/path"
)

// FileSystemStore is a structure that represents a file system store.
// It can be used to store ConfigMaps and Secrets.
// It holds paths to the configMap and secret directories,
// and a mutex to handle concurrent access.
type (
	FileSystemStore struct {
		configMapPath string
		secretPath    string
		mutex         sync.RWMutex
		logger        *zap.SugaredLogger
	}
)

// FileSystemStoreOptions represents options used to create a new FileSystemStore.
type FileSystemStoreOptions struct {
	DataPath string
}

// NewFileSystemStore initializes a new FileSystemStore with specified options.
//
// Parameters:
//   - logger: A pointer to a zap.SugaredLogger for logging purposes.
//   - opts: A FileSystemStoreOptions struct containing the configuration options for the store.
//
// Returns:
//   - *FileSystemStore: A pointer to the newly created FileSystemStore.
//   - error: An error if any occurred while creating the directories for storing ConfigMaps and secrets.
//
// This function attempts to create necessary directories at the paths specified in the FileSystemStoreOptions.
// It will create a directory for ConfigMaps and another for secrets.
// If the function encounters any errors while creating these directories, it returns an error.
func NewFileSystemStore(logger *zap.SugaredLogger, opts FileSystemStoreOptions) (*FileSystemStore, error) {
	folders := []string{ConfigMapFolder, SecretFolder}

	for _, folder := range folders {
		err := filesystem.CreateDir(path.Join(opts.DataPath, folder))
		if err != nil {
			return nil, fmt.Errorf("unable to create directory %s: %w", folder, err)
		}
	}

	return &FileSystemStore{
		configMapPath: path.Join(opts.DataPath, ConfigMapFolder),
		secretPath:    path.Join(opts.DataPath, SecretFolder),
		mutex:         sync.RWMutex{},
		logger:        logger,
	}, nil
}
