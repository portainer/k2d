package volume

import (
	"context"
	"crypto/rand"
	"fmt"
	"io"
	"path/filepath"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/portainer/k2d/pkg/filesystem"
	"go.uber.org/zap"
)

const (
	// ResourceTypeLabelKey is the key used to store the associated Kubernetes resource type in the volume labels
	// It is used to identify the type of resource that the volume is associated with such as a ConfigMap or a Secret
	ResourceTypeLabelKey = "store.k2d.io/volume/resource-type"

	// SecretTypeLabelKey is the key used to store the type of Secret in the volume labels
	// It is used to identify the type of Secret that the volume is associated with such as Opaque, kubernetes.io/dockerconfigjson, etc...
	SecretTypeLabelKey = "store.k2d.io/volume/secret-type"

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

	// RegistrySecretResourceType is the label value used to identify a volume that is associated to a Secret resource
	// used to store a Docker registry credentials.
	// It is stored on a volume as a label and used to filter volumes when listing volumes associated to Secrets
	RegistrySecretResourceType = "registrysecret"

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

// EncryptionKeyFileName is the name of the file used to store the encryption key on disk
const EncryptionKeyFileName = "volume-encryption.key"

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
	secretKind    string
	encryptionKey []byte
}

// VolumeStoreOptions represents options used to create a new VolumeStore.
type VolumeStoreOptions struct {
	DockerCli     *client.Client
	CopyImageName string
	EncryptionKey []byte
	SecretKind    string
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
		encryptionKey: opts.EncryptionKey,
		secretKind:    opts.SecretKind,
	}, nil
}

// GenerateOrRetrieveEncryptionKey generates a new encryption key or retrieves an existing one from a specified folder.
// It first checks if an encryption key file already exists in the given folder. If so, it reads the key from the file.
// Otherwise, it generates a new 32-byte encryption key, saves it to a file in the specified folder, and then returns it.
//
// Parameters:
// - logger: A pointer to a zap.SugaredLogger for logging informational messages.
// - encryptionKeyFolder: The folder where the encryption key file should be stored or retrieved from.
//
// Returns:
// - A byte slice containing the encryption key.
// - An error if any operation (key generation, file read/write, etc.) fails.
func GenerateOrRetrieveEncryptionKey(logger *zap.SugaredLogger, encryptionKeyFolder string) ([]byte, error) {
	encryptionKeyPath := filepath.Join(encryptionKeyFolder, EncryptionKeyFileName)

	keyFileExists, err := filesystem.FileExists(encryptionKeyPath)
	if err != nil {
		return nil, fmt.Errorf("unable to check if encryption key file exists: %w", err)
	}

	if keyFileExists {
		logger.Infof("encryption key file already exists. Retrieving encryption key from file: %s", encryptionKeyPath)

		key, err := filesystem.ReadFileAsString(encryptionKeyPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read encryption key: %w", err)
		}

		return []byte(key), nil
	}

	logger.Infof("encryption key file does not exist. Generating a new encryption key and storing it in file: %s", encryptionKeyPath)

	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return nil, fmt.Errorf("failed to generate encryption key: %w", err)
	}

	err = filesystem.CreateFileWithDirectories(encryptionKeyPath, key)
	if err != nil {
		return nil, fmt.Errorf("failed to write encryption key to disk: %w", err)
	}

	return key, nil
}
