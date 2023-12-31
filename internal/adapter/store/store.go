// Package store provides abstractions for interacting with Kubernetes ConfigMaps and Secrets.
//
// The package defines two primary interfaces:
// 1. SecretStore: Used for managing Kubernetes Secrets.
// 2. ConfigMapStore: Used for managing Kubernetes ConfigMaps.
//
// These interfaces encapsulate various operations such as CRUD (Create, Read, Update, Delete) operations
// on the Kubernetes Secrets and ConfigMaps, as well as generating file system binds that can be used for mounting
// files within containers.
//
// Usage Note:
//   - The method GetConfigMap() returns a 'ErrResourceNotFound' error (from the adapter/errors package) if the underlying ConfigMap resource is not found.
//   - The method GetSecret() returns a 'ErrResourceNotFound' error (from the adapter/errors package) if the underlying Secret resource is not found.
//   - The methods GetSecretBinds() and GetConfigMapBinds() are used to generate a list of filesystem binds that
//     can be used by containers for mounting files.
//
// Example:
//
// import (
//
//	adaptererr "github.com/portainer/k2d/internal/adapter/errors"
//
// )
//
// s := NewYourSecretStoreImplementation()
// secret, err := s.GetSecret("my-secret")
//
//	if err != nil {
//	   if err == adaptererr.ErrResourceNotFound {
//	      log.Println("Secret not found")
//	   } else {
//	      log.Println("An error occurred:", err)
//	   }
//	}
package store

import (
	"fmt"

	"github.com/portainer/k2d/internal/adapter/store/filesystem"
	"github.com/portainer/k2d/internal/adapter/store/memory"
	"github.com/portainer/k2d/internal/adapter/store/volume"
	"github.com/portainer/k2d/internal/types"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/kubernetes/pkg/apis/core"
)

// SecretStore is an interface for interacting with Kubernetes Secrets.
type SecretStore interface {
	DeleteSecret(secretName, namespace string) error
	GetSecretBinds(secret *core.Secret) (map[string]string, error)
	GetSecret(secretName, namespace string) (*core.Secret, error)
	GetSecrets(namespace string, selector labels.Selector) (core.SecretList, error)
	StoreSecret(secret *corev1.Secret) error
}

// ConfigMapStore is an interface for interacting with Kubernetes ConfigMaps.
type ConfigMapStore interface {
	DeleteConfigMap(configMapName, namespace string) error
	GetConfigMapBinds(configMap *core.ConfigMap) (map[string]string, error)
	GetConfigMap(configMapName, namespace string) (*core.ConfigMap, error)
	GetConfigMaps(namespace string) (core.ConfigMapList, error)
	StoreConfigMap(configMap *corev1.ConfigMap) error
}

// StoreOptions represents options that can be used to configure how to store ConfigMap and Secret resources.
// It is used by the ConfigureStore() and ConfigureRegistrySecretStore() functions to initialize and configure the storage backends.
type StoreOptions struct {
	Backend         string
	RegistryBackend string
	Logger          *zap.SugaredLogger
	Filesystem      filesystem.FileSystemStoreOptions
	Volume          volume.VolumeStoreOptions
}

// ConfigureStore initializes and configures a storage backend for ConfigMap and Secret resources based on the provided StoreOptions.
// It supports multiple backends: "disk" and "volume". For the "disk" backend, it uses a filesystem-based store.
// For the "volume" backend, it uses a volume-based store that relies on Docker volumes.
//
// Parameters:
// - opts: StoreOptions object containing configurations for initializing the storage backend.
//
// Returns:
// - ConfigMapStore: An interface for interacting with Kubernetes ConfigMaps.
// - SecretStore: An interface for interacting with Kubernetes Secrets.
// - error: An error object if any errors occur during the initialization or configuration process.
//
// Errors:
// - Returns an error if it fails to create the filesystem store for the "disk" backend.
// - Returns an error if it fails to create the volume store for the "volume" backend.
// - Returns an error if an invalid backend type is provided.
func ConfigureStore(opts StoreOptions) (ConfigMapStore, SecretStore, error) {
	switch opts.Backend {
	case types.DiskStoreBackend:
		filesystemStore, err := filesystem.NewFileSystemStore(opts.Logger, opts.Filesystem)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create filesystem store: %w", err)
		}

		opts.Logger.Info("using disk store for ConfigMaps and Secrets")
		return filesystemStore, filesystemStore, nil
	case types.VolumeStoreBackend:
		opts.Volume.SecretKind = volume.SecretResourceType
		volumeStore, err := volume.NewVolumeStore(opts.Logger, opts.Volume)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create volume store: %w", err)
		}

		opts.Logger.Info("using volume store for ConfigMaps and Secrets")
		return volumeStore, volumeStore, nil
	default:
		return nil, nil, fmt.Errorf("invalid store backend: %s", opts.Backend)
	}
}

// ConfigureRegistrySecretStore initializes and configures a storage backend for Registry Secrets based on the provided StoreOptions.
// It supports multiple backends: "memory" and "volume". For the "memory" backend, it uses an in-memory store.
// The "volume" backend utilizes a volume-based store, which relies on Docker volumes and an encryption key to store encrypted data.
//
// Parameters:
// - opts: StoreOptions object containing configurations for initializing the storage backend.
// - encryptionKeyFolder: A string representing the folder where the encryption key is stored or will be generated.
//
// Returns:
// - SecretStore: An interface for interacting with Kubernetes Registry Secrets.
// - error: An error object if any errors occur during the initialization or configuration process.
//
// Errors:
// - Returns an error if it fails to generate the encryption key for the "volume" backend.
// - Returns an error if it fails to create the volume store for the "volume" backend.
// - Returns an error if an invalid registry secret store backend is provided.
func ConfigureRegistrySecretStore(opts StoreOptions, encryptionKeyFolder string) (SecretStore, error) {
	switch opts.RegistryBackend {
	case types.MemoryRegistryStoreBackend:
		opts.Logger.Info("using memory store for registry Secrets")
		return memory.NewInMemoryStore(), nil
	case types.VolumeRegistryStoreBackend:
		opts.Logger.Info("using encrypted volume store for registry Secrets")

		encryptionKey, err := volume.GenerateOrRetrieveEncryptionKey(opts.Logger, encryptionKeyFolder)
		if err != nil {
			return nil, fmt.Errorf("failed to generate encryption key: %w", err)
		}

		opts.Volume.EncryptionKey = encryptionKey
		opts.Volume.SecretKind = volume.RegistrySecretResourceType

		volumeStore, err := volume.NewVolumeStore(opts.Logger, opts.Volume)
		if err != nil {
			return nil, fmt.Errorf("failed to create volume store: %w", err)
		}

		return volumeStore, nil
	default:
		return nil, fmt.Errorf("invalid registry secret store backend: %s", opts.RegistryBackend)
	}
}
