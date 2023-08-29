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
//   - The method GetConfigMap() returns a 'ErrResourceNotFound' error (from the errors package) if the underlying ConfigMap resource is not found.
//   - The methods GetSecretBinds() and GetConfigMapBinds() are used to generate a list of filesystem binds that
//     can be used by containers for mounting files.
//
// Example:
//
// import (
//
//	storeerr "github.com/portainer/k2d/internal/adapter/store/errors"
//
// )
//
// s := NewYourSecretStoreImplementation()
// secret, err := s.GetSecret("my-secret")
//
//	if err != nil {
//	   if err == storeerr.ErrResourceNotFound {
//	      log.Println("Secret not found")
//	   } else {
//	      log.Println("An error occurred:", err)
//	   }
//	}
package store

import (
	"fmt"

	"github.com/portainer/k2d/internal/adapter/store/filesystem"
	"github.com/portainer/k2d/internal/adapter/store/volume"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/kubernetes/pkg/apis/core"
)

// SecretStore is an interface for interacting with Kubernetes Secrets.
type SecretStore interface {
	DeleteSecret(secretName string) error
	GetSecretBinds(secret *core.Secret) ([]string, error)
	GetSecret(secretName string) (*core.Secret, error)
	GetSecrets(selector labels.Selector) (core.SecretList, error)
	StoreSecret(secret *corev1.Secret) error
}

// ConfigMapStore is an interface for interacting with Kubernetes ConfigMaps.
type ConfigMapStore interface {
	DeleteConfigMap(configMapName string) error
	GetConfigMapBinds(configMap *core.ConfigMap) ([]string, error)
	GetConfigMap(configMapName string) (*core.ConfigMap, error)
	GetConfigMaps() (core.ConfigMapList, error)
	StoreConfigMap(configMap *corev1.ConfigMap) error
}

// TODO: comments

type StoreOptions struct {
	Backend string
	Logger  *zap.SugaredLogger

	Filesystem filesystem.FileSystemStoreOptions
	Volume     volume.VolumeStoreOptions
}

// TODO: rename
func InitStoreBackend(opts StoreOptions) (ConfigMapStore, SecretStore, error) {
	switch opts.Backend {
	case "disk":
		filesystemStore, err := filesystem.NewFileSystemStore(opts.Filesystem)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create filesystem store: %w", err)
		}

		return filesystemStore, filesystemStore, nil
	case "volume":
		volumeStore, err := volume.NewVolumeStore(opts.Logger, opts.Volume)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create volume store: %w", err)
		}

		return volumeStore, volumeStore, nil
	default:
		return nil, nil, fmt.Errorf("invalid store backend: %s", opts.Backend)
	}
}
