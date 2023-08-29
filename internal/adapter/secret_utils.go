package adapter

import (
	"fmt"

	"github.com/portainer/k2d/internal/adapter/types"
	"github.com/portainer/k2d/pkg/filesystem"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TODO: comments
func (adapter *KubeDockerAdapter) StoreSystemSecret(tokenPath, caPath string) error {
	token, err := filesystem.ReadFileAsString(tokenPath)
	if err != nil {
		return fmt.Errorf("failed to read token file: %w", err)
	}

	ca, err := filesystem.ReadFileAsString(caPath)
	if err != nil {
		return fmt.Errorf("failed to read ca file: %w", err)
	}

	secret := corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        types.SystemSecretName,
			Annotations: map[string]string{},
			Namespace:   "default",
		},
		StringData: map[string]string{
			"token":  token,
			"ca.crt": ca,
		},
	}

	return adapter.secretStore.StoreSecret(&secret)
}
