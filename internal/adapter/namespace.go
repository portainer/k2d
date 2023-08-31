package adapter

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/errdefs"
	adaptererr "github.com/portainer/k2d/internal/adapter/errors"
	k2dtypes "github.com/portainer/k2d/internal/adapter/types"
	"github.com/portainer/k2d/internal/k8s"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubernetes/pkg/apis/core"
)

func (adapter *KubeDockerAdapter) CreateNetworkFromNamespace(ctx context.Context, namespace *corev1.Namespace) error {
	networkName := buildNetworkName(namespace.Name)

	network, err := adapter.getNetwork(ctx, networkName)
	if err != nil && !errors.Is(err, adaptererr.ErrResourceNotFound) {
		return fmt.Errorf("unable to check for network existence: %w", err)
	}

	if network != nil {
		return fmt.Errorf("network %s already exists", networkName)
	}

	if namespace.Labels["app.kubernetes.io/managed-by"] == "Helm" {
		namespaceData, err := json.Marshal(namespace)
		if err != nil {
			return fmt.Errorf("unable to marshal deployment: %w", err)
		}
		namespace.ObjectMeta.Annotations["kubectl.kubernetes.io/last-applied-configuration"] = string(namespaceData)
	}

	lastAppliedConfiguration := ""
	if namespace.ObjectMeta.Annotations["kubectl.kubernetes.io/last-applied-configuration"] != "" {
		lastAppliedConfiguration = namespace.ObjectMeta.Annotations["kubectl.kubernetes.io/last-applied-configuration"]
	}

	networkOptions := types.NetworkCreate{
		Driver: "bridge",
		Labels: map[string]string{
			k2dtypes.NamespaceLabelKey:                  namespace.Name,
			k2dtypes.NamespaceLastAppliedConfigLabelKey: lastAppliedConfiguration,
		},
		Options: map[string]string{
			"com.docker.network.bridge.name": networkName,
		},
	}

	_, err = adapter.cli.NetworkCreate(ctx, networkName, networkOptions)
	if err != nil {
		return fmt.Errorf("unable to create network %s: %w", networkName, err)
	}

	return nil
}

func (adapter *KubeDockerAdapter) DeleteNamespace(ctx context.Context, namespaceName string) error {
	labelFilter := filters.NewArgs()
	labelFilter.Add("label", fmt.Sprintf("%s=%s", k2dtypes.NamespaceLabelKey, namespaceName))

	containers, err := adapter.cli.ContainerList(ctx, types.ContainerListOptions{All: true, Filters: labelFilter})
	if err != nil {
		return fmt.Errorf("unable to list containers: %w", err)
	}

	for _, container := range containers {
		adapter.DeleteContainer(ctx, container.Names[0], namespaceName)
	}

	// This is just to make sure that the containers have been properly deleted
	// before we try to delete the network
	time.Sleep(3 * time.Second)

	networkName := buildNetworkName(namespaceName)
	err = adapter.cli.NetworkRemove(ctx, networkName)
	if err != nil {
		return fmt.Errorf("unable to delete network %s: %w", networkName, err)
	}

	return nil
}

func (adapter *KubeDockerAdapter) GetNamespace(ctx context.Context, namespaceName string) (*corev1.Namespace, error) {
	networkName := buildNetworkName(namespaceName)

	network, err := adapter.getNetwork(ctx, networkName)
	if err != nil {
		return nil, fmt.Errorf("unable to get namespace %s: %w", namespaceName, err)
	}

	versionedNamespace := corev1.Namespace{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Namespace",
			APIVersion: "v1",
		},
	}

	namespace := adapter.converter.ConvertNetworkToNamespace(namespaceName, *network)

	err = adapter.ConvertK8SResource(&namespace, &versionedNamespace)
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

func (adapter *KubeDockerAdapter) getNetwork(ctx context.Context, networkName string) (*types.NetworkResource, error) {
	network, err := adapter.cli.NetworkInspect(ctx, networkName, types.NetworkInspectOptions{})
	if err != nil {
		if errdefs.IsNotFound(err) {
			return nil, adaptererr.ErrResourceNotFound
		}
		return nil, fmt.Errorf("unable to inspect network %s: %w", networkName, err)
	}

	return &network, nil
}

func (adapter *KubeDockerAdapter) listNamespaces(ctx context.Context) (core.NamespaceList, error) {
	labelFilter := filters.NewArgs()
	labelFilter.Add("label", k2dtypes.NamespaceLabelKey)

	networks, err := adapter.cli.NetworkList(ctx, types.NetworkListOptions{Filters: labelFilter})
	if err != nil {
		return core.NamespaceList{}, fmt.Errorf("unable to list networks: %w", err)
	}

	namespaceList := []core.Namespace{}

	for _, network := range networks {
		namespace := network.Labels[k2dtypes.NamespaceLabelKey]
		namespaceList = append(namespaceList, adapter.converter.ConvertNetworkToNamespace(namespace, network))
	}

	return core.NamespaceList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "NamespaceList",
			APIVersion: "v1",
		},

		Items: namespaceList,
	}, nil
}
