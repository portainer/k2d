package filesystem

import (
	"fmt"
	"path"
	"sync"

	"github.com/portainer/k2d/pkg/filesystem"
)

// Constants representing folder names and separators used in file paths.
const (
	CONFIGMAP_FOLDER    = "configmaps"
	SECRET_FOLDER       = "secrets"
	CONFIGMAP_SEPARATOR = "-k2dcm-"
	SECRET_SEPARATOR    = "-k2dsec-"
)

// FileSystemStore is a structure that represents a file system store.
// It can be used to store ConfigMaps and Secrets.
// It holds paths to the configMap and secret directories,
// and a mutex to handle concurrent access.
type (
	FileSystemStore struct {
		configMapPath string
		secretPath    string
		mutex         sync.Mutex
	}
)

// NewFileSystemStore creates and returns a new FileSystemStore.
// It receives a data path where the directories for configMaps and secrets are created.
// If the directories cannot be created, an error is returned.
func NewFileSystemStore(dataPath string) (*FileSystemStore, error) {
	folders := []string{CONFIGMAP_FOLDER, SECRET_FOLDER}

	for _, folder := range folders {
		err := filesystem.CreateDir(path.Join(dataPath, folder))
		if err != nil {
			return nil, fmt.Errorf("unable to create directory %s: %w", folder, err)
		}
	}

	return &FileSystemStore{
		configMapPath: path.Join(dataPath, CONFIGMAP_FOLDER),
		secretPath:    path.Join(dataPath, SECRET_FOLDER),
		mutex:         sync.Mutex{},
	}, nil
}
