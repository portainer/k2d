package adapter

import (
	"context"
	"fmt"

	k2dtypes "github.com/portainer/k2d/internal/adapter/types"
	"github.com/portainer/k2d/internal/k8s"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubernetes/pkg/apis/core"
)

func (adapter *KubeDockerAdapter) CreateNetworkFromNamespace(ctx context.Context, namespace *corev1.Namespace) error {
	err := adapter.CreateNetworkFromNamespaceSpec(ctx, *namespace)
	if err != nil {
		return fmt.Errorf("unable to create network from namespace spec: %w", err)
	}

	return nil
}

func (adapter *KubeDockerAdapter) ListNamespaces(ctx context.Context) (corev1.NamespaceList, error) {
	namespaceList, err := adapter.listNamespaces(ctx)
	if err != nil {
		return corev1.NamespaceList{}, fmt.Errorf("unable to list namespaces: %w", err)
	}

	versionedNamespaceList := corev1.NamespaceList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "NamespaceList",
			APIVersion: "v1",
		},
	}

	err = adapter.ConvertK8SResource(&namespaceList, &versionedNamespaceList)
	if err != nil {
		return corev1.NamespaceList{}, fmt.Errorf("unable to convert internal NamespaceList to versioned NamespaceList: %w", err)
	}

	return versionedNamespaceList, nil
}

func (adapter *KubeDockerAdapter) GetNamespace(ctx context.Context, namespaceName string) (*corev1.Namespace, error) {
	network, err := adapter.GetNetwork(ctx, namespaceName)
	if err != nil {
		return &corev1.Namespace{}, fmt.Errorf("unable to get the namespaces: %w", err)
	}

	if network == nil {
		return nil, nil
	}

	versionedNamespace := corev1.Namespace{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Namespace",
			APIVersion: "v1",
		},
	}

	namespace := adapter.converter.ConvertNetworkToNamespace(network)

	err = adapter.ConvertK8SResource(namespace, &versionedNamespace)
	if err != nil {
		return nil, fmt.Errorf("unable to convert internal object to versioned object: %w", err)
	}

	return &versionedNamespace, nil
}

func (adapter *KubeDockerAdapter) GetNamespaceTable(ctx context.Context) (*metav1.Table, error) {
	namespaceList, err := adapter.listNamespaces(ctx)
	if err != nil {
		return &metav1.Table{}, fmt.Errorf("unable to list namespaces: %w", err)
	}
	return k8s.GenerateTable(&namespaceList)
}

func (adapter *KubeDockerAdapter) listNamespaces(ctx context.Context) (core.NamespaceList, error) {
	networks, err := adapter.ListNetworks(ctx)
	if err != nil {
		adapter.logger.Errorf("unable to list networks: %v", err)
		return core.NamespaceList{}, err
	}

	namespaceList := []core.Namespace{}

	for _, network := range networks {
		if network.Labels[k2dtypes.NamespaceLabelKey] != "" {
			namespaceList = append(namespaceList, *adapter.converter.ConvertNetworkToNamespace(&network))
		}
	}
	return core.NamespaceList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "NamespaceList",
			APIVersion: "v1",
		},

		Items: namespaceList,
	}, nil
}

func (adapter *KubeDockerAdapter) DeleteNamespace(ctx context.Context, namespaceName string) error {
	err := adapter.DeleteNetwork(ctx, namespaceName)
	if err != nil {
		return fmt.Errorf("unable to delete the namespaces: %w", err)
	}

	return nil
}
