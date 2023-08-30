package adapter

import (
	"fmt"

	"github.com/portainer/k2d/internal/k8s"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubernetes/pkg/apis/core"
)

func (adapter *KubeDockerAdapter) CreateConfigMap(configMap *corev1.ConfigMap) error {
	return adapter.configMapStore.StoreConfigMap(configMap)
}

func (adapter *KubeDockerAdapter) DeleteConfigMap(configMapName, namespace string) error {
	return adapter.configMapStore.DeleteConfigMap(configMapName, namespace)
}

func (adapter *KubeDockerAdapter) GetConfigMap(configMapName, namespace string) (*corev1.ConfigMap, error) {
	configMap, err := adapter.configMapStore.GetConfigMap(configMapName, namespace)
	if err != nil {
		return nil, fmt.Errorf("unable to get configmap: %w", err)
	}

	versionedConfigMap := corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
	}

	err = adapter.ConvertK8SResource(configMap, &versionedConfigMap)
	if err != nil {
		return nil, fmt.Errorf("unable to convert internal object to versioned object: %w", err)
	}

	versionedConfigMap.ObjectMeta.Annotations["kubectl.kubernetes.io/last-applied-configuration"] = ""

	return &versionedConfigMap, nil
}

func (adapter *KubeDockerAdapter) GetConfigMapTable(namespace string) (*metav1.Table, error) {
	configMapList, err := adapter.listConfigMaps(namespace)
	if err != nil {
		return &metav1.Table{}, fmt.Errorf("unable to list configmaps: %w", err)
	}

	return k8s.GenerateTable(&configMapList)
}

func (adapter *KubeDockerAdapter) ListConfigMaps(namespace string) (corev1.ConfigMapList, error) {
	configMapList, err := adapter.listConfigMaps(namespace)
	if err != nil {
		return corev1.ConfigMapList{}, fmt.Errorf("unable to list configmaps: %w", err)
	}

	versionedConfigMapList := corev1.ConfigMapList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMapList",
			APIVersion: "v1",
		},
	}

	err = adapter.ConvertK8SResource(&configMapList, &versionedConfigMapList)
	if err != nil {
		return corev1.ConfigMapList{}, fmt.Errorf("unable to convert internal ConfigMapList to versioned ConfigMapList: %w", err)
	}

	return versionedConfigMapList, nil
}

func (adapter *KubeDockerAdapter) listConfigMaps(namespace string) (core.ConfigMapList, error) {
	return adapter.configMapStore.GetConfigMaps(namespace)
}
