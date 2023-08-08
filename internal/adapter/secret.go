package adapter

import (
	"fmt"

	"github.com/portainer/k2d/internal/k8s"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
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

	err = adapter.ConvertK8SResource(secret, &versionedSecret)
	if err != nil {
		return nil, fmt.Errorf("unable to convert internal object to versioned object: %w", err)
	}

	versionedSecret.ObjectMeta.Annotations["kubectl.kubernetes.io/last-applied-configuration"] = ""

	return &versionedSecret, nil
}

func (adapter *KubeDockerAdapter) ListSecrets(selector labels.Selector) (corev1.SecretList, error) {

	secretList, err := adapter.listSecrets(selector)
	if err != nil {
		return corev1.SecretList{}, fmt.Errorf("unable to list secrets: %w", err)
	}

	versionedSecretList := corev1.SecretList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "SecretList",
			APIVersion: "v1",
		},
	}

	err = adapter.ConvertK8SResource(&secretList, &versionedSecretList)
	if err != nil {
		return corev1.SecretList{}, fmt.Errorf("unable to convert internal SecretList to versioned SecretList: %w", err)
	}

	return versionedSecretList, nil
}

func (adapter *KubeDockerAdapter) GetSecretTable(selector labels.Selector) (*metav1.Table, error) {
	secretList, err := adapter.listSecrets(selector)
	if err != nil {
		return &metav1.Table{}, fmt.Errorf("unable to list secrets: %w", err)
	}

	return k8s.GenerateTable(&secretList)
}

func (adapter *KubeDockerAdapter) listSecrets(selector labels.Selector) (core.SecretList, error) {
	return adapter.fileSystemStore.GetSecrets(selector)
}
