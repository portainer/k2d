package adapter

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubernetes/pkg/apis/core"
)

func (adapter *KubeDockerAdapter) CreateConfigMap(configMap *corev1.ConfigMap) error {
	return adapter.fileSystemStore.StoreConfigMap(configMap)
}

func (adapter *KubeDockerAdapter) DeleteConfigMap(configMapName string) error {
	return adapter.fileSystemStore.DeleteConfigMap(configMapName)
}

func (adapter *KubeDockerAdapter) GetConfigMap(configMapName string) (*corev1.ConfigMap, error) {
	configMap, err := adapter.fileSystemStore.GetConfigMap(configMapName)
	if err != nil {
		return &corev1.ConfigMap{}, fmt.Errorf("unable to get configmap: %w", err)
	}

	versionedConfigMap := corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
	}

	err = adapter.ConvertObjectToVersionedObject(configMap, &versionedConfigMap)
	if err != nil {
		return nil, fmt.Errorf("unable to convert object to versioned object: %w", err)
	}

	versionedConfigMap.ObjectMeta.Annotations["kubectl.kubernetes.io/last-applied-configuration"] = ""

	return &versionedConfigMap, nil
}

func (adapter *KubeDockerAdapter) ListConfigMaps() (core.ConfigMapList, error) {
	return adapter.fileSystemStore.GetConfigMaps()
}
