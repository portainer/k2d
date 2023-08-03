// Package filesystem provides functionality to interact with the file system.
package filesystem

import (
	"sync"
)

// Constants representing folder names and separators used in file paths.
const (
	CONFIGMAP_FOLDER    = "configmaps"
	SECRET_FOLDER       = "secrets"
	CONFIGMAP_SEPARATOR = "-k2dcm-"
	SECRET_SEPARATOR    = "-k2dsec-"
)

// FileSystemStore is a structure that represents a file system store.
// It holds paths to the configMap and secret directories,
// and a mutex to handle concurrent access.
type (
	FileSystemStore struct {
		path  string
		mutex sync.Mutex
	}
)

// NewFileSystemStore creates and returns a new FileSystemStore.
// It receives a data path where the directories for configMaps and secrets are created.
// If the directories cannot be created, an error is returned.
func NewFileSystemStore(dataPath string) (*FileSystemStore, error) {
	return &FileSystemStore{
		path:  dataPath,
		mutex: sync.Mutex{},
	}, nil
}
