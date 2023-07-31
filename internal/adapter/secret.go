package adapter

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubernetes/pkg/apis/core"
)

func (adapter *KubeDockerAdapter) CreateSecret(secret *corev1.Secret) error {
	return adapter.fileSystemStore.StoreSecret(secret)
}

func (adapter *KubeDockerAdapter) DeleteSecret(secretName string) error {
	return adapter.fileSystemStore.DeleteSecret(secretName)
}

func (adapter *KubeDockerAdapter) GetSecret(secretName string) (*corev1.Secret, error) {
	secret, err := adapter.fileSystemStore.GetSecret(secretName)
	if err != nil {
		return &corev1.Secret{}, fmt.Errorf("unable to get secret: %w", err)
	}

	versionedSecret := corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
	}

	err = adapter.ConvertObjectToVersionedObject(secret, &versionedSecret)
	if err != nil {
		return nil, fmt.Errorf("unable to convert object to versioned object: %w", err)
	}

	versionedSecret.ObjectMeta.Annotations["kubectl.kubernetes.io/last-applied-configuration"] = ""

	return &versionedSecret, nil
}

func (adapter *KubeDockerAdapter) ListSecrets() (core.SecretList, error) {
	return adapter.fileSystemStore.GetSecrets()
}
