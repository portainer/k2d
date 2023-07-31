// Package converter contains types and functions for converting Kubernetes objects to Docker objects and vice versa.
package converter

import (
	"github.com/portainer/k2d/internal/adapter/filesystem"
	"github.com/portainer/k2d/internal/types"
)

// DockerAPIConverter is a struct that facilitates the conversion of Kubernetes objects into Docker API compatible configurations.
// It contains a FileSystemStore for accessing data from the filesystem as well as the k2dServerAddr and k2dServerPort which will be shared with all
// created containers.
type DockerAPIConverter struct {
	store                  *filesystem.FileSystemStore
	k2dServerConfiguration *types.K2DServerConfiguration
}

// NewDockerAPIConverter creates and returns a new DockerAPIConverter.
// It receives a FileSystemStore which is used for accessing data from the filesystem.
func NewDockerAPIConverter(store *filesystem.FileSystemStore, k2dServerConfig *types.K2DServerConfiguration) *DockerAPIConverter {
	return &DockerAPIConverter{
		store:                  store,
		k2dServerConfiguration: k2dServerConfig,
	}
}
