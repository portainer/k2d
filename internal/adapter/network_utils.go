package adapter

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	k2dtypes "github.com/portainer/k2d/internal/adapter/types"
	corev1 "k8s.io/api/core/v1"
)

// EnsureRequiredDockerResourcesExist verifies the existence of required Docker resources
// for k2d to work. If the required resources do not exist, it will attempt to create them.
//
// Specifically, this function checks for the existence of a Docker network with a specific name
// (K2DNetworkName). If this network does not exist, the function attempts to create it.
//
// It returns an error if it fails to list Docker networks or if it fails to create the required Docker network.
func (adapter *KubeDockerAdapter) EnsureRequiredDockerResourcesExist(ctx context.Context) error {
	k2dNetwork, err := adapter.GetNetwork(ctx, k2dtypes.K2DNetworkName)
	if err != nil {
		return fmt.Errorf("unable to list networks: %w", err)
	}

	if k2dNetwork == nil {
		adapter.logger.Info("creating k2d container network")
		_, err := adapter.cli.NetworkCreate(ctx, k2dtypes.K2DNetworkName, types.NetworkCreate{
			Labels: map[string]string{
				k2dtypes.NamespaceLabelKey: "default",
			},
		})
		if err != nil {
			return fmt.Errorf("unable to create k2d container network: %w", err)
		}

	} else {
		adapter.logger.Info("k2d container network already exists, skipping creation")
	}

	return nil
}

// GetNetwork searches through the provided slice of NetworkResource types,
// looking for a network resource that matches the provided networkName.
// If a matching network resource is found, it returns a pointer to it.
// If no match is found, it returns nil.
func (adapter *KubeDockerAdapter) GetNetwork(ctx context.Context, networkName string) (*types.NetworkResource, error) {
	if networkName == "default" {
		networkName = "k2d_net"
	}

	labelFilter := filters.NewArgs()
	labelFilter.Add("name", networkName)
	network, err := adapter.cli.NetworkList(ctx, types.NetworkListOptions{Filters: labelFilter})
	if err != nil {
		return &types.NetworkResource{}, fmt.Errorf("unable to list networks: %w", err)
	}

	if len(network) == 0 {
		return nil, nil
	}

	return &network[0], nil
}

// ListNetwork returns a list of Docker networks.
// It returns an error if it fails to list the networks.
func (adapter *KubeDockerAdapter) ListNetworks(ctx context.Context) ([]types.NetworkResource, error) {
	networks, err := adapter.cli.NetworkList(ctx, types.NetworkListOptions{})
	if err != nil {
		return nil, fmt.Errorf("unable to list networks: %w", err)
	}

	return networks, nil
}

// CreateNetworkFromNamespace creates a Docker network with the provided name.
// It returns an error if it fails to create the network.
func (adapter *KubeDockerAdapter) CreateNetworkFromNamespaceSpec(ctx context.Context, namespace corev1.Namespace) error {
	adapter.logger.Infof("creating network %s", namespace.Name)
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

	_, err := adapter.cli.NetworkCreate(ctx, namespace.Name, types.NetworkCreate{
		Driver: "bridge",
		Labels: map[string]string{
			k2dtypes.NamespaceLabelKey:                  namespace.Name,
			k2dtypes.NamespaceLastAppliedConfigLabelKey: lastAppliedConfiguration,
		},
	})
	if err != nil {
		return fmt.Errorf("unable to create network %s: %w", namespace.Name, err)
	}

	return nil
}

// DeleteNetwork deletes a Docker network with the provided name.
// It returns an error if it fails to delete the network.
func (adapter *KubeDockerAdapter) DeleteNetwork(ctx context.Context, networkName string) error {
	adapter.logger.Infof("deleting network %s", networkName)
	err := adapter.cli.NetworkRemove(ctx, networkName)
	if err != nil {
		return fmt.Errorf("unable to delete network %s: %w", networkName, err)
	}

	return nil
}
