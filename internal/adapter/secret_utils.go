package adapter

import (
	"fmt"

	"github.com/portainer/k2d/internal/adapter/types"
	"github.com/portainer/k2d/pkg/filesystem"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// StoreServiceAccountSecret takes the paths of a service account token file and a CA certificate file,
// reads their content, and stores them as a new Kubernetes Secret object. This function is specifically
// designed to handle the system service account secret, which is used to authenticate with the Kubernetes
// API server.
//
// Parameters:
//   - tokenPath: The file path where the service account token is stored.
//   - caPath: The file path where the CA certificate is stored.
//
// Returns:
//   - nil if the secret is successfully stored.
//   - an error if reading the token or CA certificate files fails, or if storing the secret fails.
func (adapter *KubeDockerAdapter) StoreServiceAccountSecret(tokenPath, caPath string) error {
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
			Name:        types.K2dServiceAccountSecretName,
			Annotations: map[string]string{},
			Namespace:   "default",
		},
		Type: corev1.SecretTypeOpaque,
		StringData: map[string]string{
			"token":  token,
			"ca.crt": ca,
		},
	}

	return adapter.secretStore.StoreSecret(&secret)
}
