package volume

import (
	"context"
	"fmt"
	"io"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"go.uber.org/zap"
)

const (
	// ResourceTypeLabelKey is the key used to store the associated Kubernetes resource type in the volume labels
	// It is used to identify the type of resource that the volume is associated with such as a ConfigMap or a Secret
	ResourceTypeLabelKey = "store.k2d.io/volume/resource-type"

	// VolumeNameLabelKey is the key used to store the name of a volume in the resource labels
	// It is used to identify the name of the volume associated with a ConfigMap or a Secret
	VolumeNameLabelKey = "store.k2d.io/volume/volume-name"
)

const (
	// ConfigMapVolumePrefix is the prefix used to name volumes associated to ConfigMap resources
	// A prefix is used to avoid clash with Secret volumes
	ConfigMapVolumePrefix = "configmap-"

	// ConfigMapResourceType is the label value used to identify a volume that is associated to a ConfigMap resource
	// It is stored on a volume as a label and used to filter volumes when listing volumes associated to ConfigMaps
	ConfigMapResourceType = "configmap"

	// SecretVolumePrefix is the prefix used to name volumes associated to Secret resources
	// A prefix is used to avoid clash with ConfigMap volumes
	SecretVolumePrefix = "secret-"

	// SecretResourceType is the label value used to identify a volume that is associated to a Secret resource
	// It is stored on a volume as a label and used to filter volumes when listing volumes associated to Secrets
	SecretResourceType = "secret"

	// WorkingDirName is the name of the working directory used to store data in a volume
	// It should be available at the root / inside the copy container
	WorkingDirName = "/work"
)

// VolumeStore provides an implementation of the SecretStore and ConfigMapStore interfaces,
// leveraging Docker volumes to store the contents of Kubernetes Secrets and ConfigMaps.
//
// It uses ephemeral lightweight containers to copy and read data to and from Docker volumes.
// It includes two fields:
// - cli: A Docker client used to interact with the Docker engine.
// - logger: A logger to output logs.
type VolumeStore struct {
	cli           *client.Client
	logger        *zap.SugaredLogger
	copyImageName string
}

// VolumeStoreOptions represents options used to create a new VolumeStore.
type VolumeStoreOptions struct {
	DockerCli     *client.Client
	CopyImageName string
}

// NewVolumeStore creates a new instance of VolumeStore.
//
// The function attempts to pull a specific Docker image (defined by the CopyImageName constant)
// that will be used for ephemeral containers responsible for copying and reading data.
// If the image pulling fails, the function returns an error.
//
// Parameters:
// - cli: A Docker client used to interact with the Docker engine.
// - logger: A logger to output logs.
//
// Returns:
// - A pointer to the created VolumeStore instance.
// - An error if any occurs during the initialization, like failing to pull the copy image.
func NewVolumeStore(logger *zap.SugaredLogger, opts VolumeStoreOptions) (*VolumeStore, error) {
	out, err := opts.DockerCli.ImagePull(context.TODO(), opts.CopyImageName, types.ImagePullOptions{})
	if err != nil {
		return nil, fmt.Errorf("unable to pull volume copy image: %w", err)
	}
	defer out.Close()
	io.Copy(io.Discard, out)

	return &VolumeStore{
		cli:           opts.DockerCli,
		logger:        logger,
		copyImageName: opts.CopyImageName,
	}, nil
}
