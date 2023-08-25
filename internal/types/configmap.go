package types

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/kubernetes/pkg/apis/core"
)

// ConfigMapStore is an interface for interacting with Kubernetes ConfigMaps.
type ConfigMapStore interface {
	DeleteConfigMap(configMapName string) error
	GetConfigMap(configMapName string) (*core.ConfigMap, error)
	GetConfigMaps() (core.ConfigMapList, error)
	StoreConfigMap(configMap *corev1.ConfigMap) error
}
