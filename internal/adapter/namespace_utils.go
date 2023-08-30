package adapter

import (
	"context"
	"errors"
	"fmt"

	adaptererr "github.com/portainer/k2d/internal/adapter/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// provisionNamespace provisions a Kubernetes namespace and its corresponding Docker network.
//
// The function performs the following steps:
// 1. Checks if the given namespace already exists by calling the GetNamespace method.
// 2. If the namespace exists, logs a message indicating its existence and skips the creation process.
// 3. If the namespace does not exist or an error occurs while checking its existence, proceeds to create it.
// 4. Logs a message indicating the creation of the new namespace.
// 5. Constructs a Kubernetes Namespace object with provided metadata.
// 6. Calls CreateNetworkFromNamespace to provision the Docker network corresponding to the new namespace.
//
// Parameters:
// - ctx: The context within which the function operates. Used for timeout and cancellation signals.
// - namespaceName: The name of the namespace to be provisioned.
//
// Returns:
//   - An error if any step in the process fails. This could be due to issues such as network creation failures,
//     Kubernetes API errors, or problems in checking for the existence of the namespace.
//
// Notes:
// - The function logs informative messages to indicate the steps it's performing.
// - The function is idempotent; if the namespace already exists, it will not attempt to create it again.
func (adapter *KubeDockerAdapter) provisionNamespace(ctx context.Context, namespaceName string) error {
	networkExists := true

	_, err := adapter.GetNamespace(ctx, namespaceName)
	if err != nil {
		if errors.Is(err, adaptererr.ErrResourceNotFound) {
			networkExists = false
		} else {
			return fmt.Errorf("unable to check for %s namespace existence: %w", namespaceName, err)
		}
	}

	if networkExists {
		adapter.logger.Infof("%s namespace already exists, skipping creation", namespaceName)
		return nil
	}

	adapter.logger.Infof("creating %s namespace", namespaceName)

	namespace := corev1.Namespace{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Namespace",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        namespaceName,
			Labels:      map[string]string{},
			Annotations: map[string]string{},
		},
	}

	err = adapter.CreateNetworkFromNamespace(ctx, &namespace)
	if err != nil {
		return fmt.Errorf("unable to create %s namespace: %w", namespaceName, err)
	}

	return nil
}
