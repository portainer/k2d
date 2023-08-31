package adapter

import (
	"errors"
	"fmt"

	adaptererr "github.com/portainer/k2d/internal/adapter/errors"
	"github.com/portainer/k2d/internal/k8s"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/kubernetes/pkg/apis/core"
)

func (adapter *KubeDockerAdapter) CreateSecret(secret *corev1.Secret) error {
	if secret.Type == corev1.SecretTypeDockerConfigJson {
		return adapter.registrySecretStore.StoreSecret(secret)
	}

	return adapter.secretStore.StoreSecret(secret)
}

func (adapter *KubeDockerAdapter) DeleteSecret(secretName, namespace string) error {
	return adapter.secretStore.DeleteSecret(secretName, namespace)
}

func (adapter *KubeDockerAdapter) GetSecret(secretName, namespace string) (*corev1.Secret, error) {
	secret, err := adapter.getSecret(secretName, namespace)
	if err != nil {
		return nil, fmt.Errorf("unable to get secret: %w", err)
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

func (adapter *KubeDockerAdapter) GetSecretTable(namespace string, selector labels.Selector) (*metav1.Table, error) {
	secretList, err := adapter.listSecrets(namespace, selector)
	if err != nil {
		return &metav1.Table{}, fmt.Errorf("unable to list secrets: %w", err)
	}

	return k8s.GenerateTable(&secretList)
}

func (adapter *KubeDockerAdapter) ListSecrets(namespace string, selector labels.Selector) (corev1.SecretList, error) {
	secretList, err := adapter.listSecrets(namespace, selector)
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

// when fetching a secret, we first try to get it from the secret store
// if it's not found, we try to get it from the registry secret store
func (adapter *KubeDockerAdapter) getSecret(secretName, namespace string) (*core.Secret, error) {
	secret, err := adapter.secretStore.GetSecret(secretName, namespace)
	if err != nil && !errors.Is(err, adaptererr.ErrResourceNotFound) {
		return nil, fmt.Errorf("unable to get secret: %w", err)
	}
	if secret != nil {
		return secret, nil
	}

	registrySecret, err := adapter.registrySecretStore.GetSecret(secretName, namespace)
	if err != nil && !errors.Is(err, adaptererr.ErrResourceNotFound) {
		return nil, fmt.Errorf("unable to get registry secret: %w", err)
	}
	if registrySecret != nil {
		return registrySecret, nil
	}

	return nil, adaptererr.ErrResourceNotFound
}

func (adapter *KubeDockerAdapter) listSecrets(namespace string, selector labels.Selector) (core.SecretList, error) {
	secretList, err := adapter.secretStore.GetSecrets(namespace, selector)
	if err != nil {
		return core.SecretList{}, fmt.Errorf("unable to list secrets: %w", err)
	}

	registrySecretList, err := adapter.registrySecretStore.GetSecrets(namespace, selector)
	if err != nil {
		return core.SecretList{}, fmt.Errorf("unable to list registry secrets: %w", err)
	}

	secretList.Items = append(secretList.Items, registrySecretList.Items...)

	return secretList, nil
}
