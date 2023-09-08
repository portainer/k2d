// Package converter contains types and functions for converting Kubernetes objects to Docker objects and vice versa.
package converter

import (
	containertypes "github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/portainer/k2d/internal/adapter/store"
	"github.com/portainer/k2d/internal/types"
	"github.com/portainer/k2d/pkg/rand"
)

// DockerAPIConverter is a struct that facilitates the conversion of Kubernetes objects into Docker API compatible configurations.
// It contains a FileSystemStore for accessing data from the filesystem as well as the k2dServerAddr and k2dServerPort which will be shared with all
// created containers.
type DockerAPIConverter struct {
	configMapStore         store.ConfigMapStore
	secretStore            store.SecretStore
	k2dServerConfiguration *types.K2DServerConfiguration
	portGenerator          *rand.PortGenerator
}

// ContainerConfiguration is a wrapper around the Docker API container configuration
type ContainerConfiguration struct {
	ContainerName   string
	ContainerState  *containertypes.ContainerState
	ContainerConfig *container.Config
	HostConfig      *container.HostConfig
	NetworkConfig   *network.NetworkingConfig
}

// NewDockerAPIConverter creates and returns a new DockerAPIConverter.
// It receives a FileSystemStore which is used for accessing data from the filesystem.
func NewDockerAPIConverter(configMapStore store.ConfigMapStore, secretStore store.SecretStore, k2dServerConfig *types.K2DServerConfiguration) *DockerAPIConverter {
	return &DockerAPIConverter{
		configMapStore:         configMapStore,
		secretStore:            secretStore,
		k2dServerConfiguration: k2dServerConfig,
		portGenerator:          rand.NewPortGenerator(),
	}
}
