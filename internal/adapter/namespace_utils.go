package adapter

import (
	"context"
	"errors"
	"fmt"

	adaptererr "github.com/portainer/k2d/internal/adapter/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TODO: comment
// Should probably be part of a larger function and not exposed
func (adapter *KubeDockerAdapter) ProvisionNamespace(ctx context.Context, namespaceName string) error {
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
