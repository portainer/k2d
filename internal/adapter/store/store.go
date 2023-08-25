package store

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/kubernetes/pkg/apis/core"
)

// SecretStore is an interface for interacting with Kubernetes Secrets.
type SecretStore interface {
	DeleteSecret(secretName string) error
	GetSecret(secretName string) (*core.Secret, error)
	GetSecrets(selector labels.Selector) (core.SecretList, error)
	StoreSecret(secret *corev1.Secret) error
}

// ConfigMapStore is an interface for interacting with Kubernetes ConfigMaps.
type ConfigMapStore interface {
	DeleteConfigMap(configMapName string) error
	GetConfigMap(configMapName string) (*core.ConfigMap, error)
	GetConfigMaps() (core.ConfigMapList, error)
	StoreConfigMap(configMap *corev1.ConfigMap) error
}
