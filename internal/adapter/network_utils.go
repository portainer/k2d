package adapter

import (
	"context"
	"fmt"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	k2dtypes "github.com/portainer/k2d/internal/adapter/types"
)

// EnsureRequiredDockerResourcesExist verifies the existence of required Docker resources
// for k2d to work. If the required resources do not exist, it will attempt to create them.
//
// Specifically, this function checks for the existence of a Docker network with a specific name
// (K2DNetworkName). If this network does not exist, the function attempts to create it.
//
// It returns an error if it fails to list Docker networks or if it fails to create the required Docker network.
func (adapter *KubeDockerAdapter) EnsureRequiredDockerResourcesExist(ctx context.Context) error {
	k2dNetwork, err := adapter.GetNetwork(ctx, "default")
	if err != nil {
		return fmt.Errorf("unable to list networks: %w", err)
	}

	if k2dNetwork != nil {
		adapter.logger.Info("k2d container network already exists, skipping creation")
	} else {
		adapter.logger.Info("creating k2d container network")
		_, err := adapter.cli.NetworkCreate(ctx, k2dtypes.K2DNetworkName, types.NetworkCreate{
			Labels: map[string]string{
				k2dtypes.NamespaceLabelKey: "default",
			},
		})
		if err != nil {
			return fmt.Errorf("unable to create k2d container network: %w", err)
		}
	}

	return nil
}

// GetNetwork searches through the provided slice of NetworkResource types,
// looking for a network resource that matches the provided networkName.
// If a matching network resource is found, it returns a pointer to it.
// If no match is found, it returns nil.
func (adapter *KubeDockerAdapter) GetNetwork(ctx context.Context, networkName string) (*types.NetworkResource, error) {
	labelFilter := filters.NewArgs()
	labelFilter.Add("label", fmt.Sprintf("%s=%s", k2dtypes.NamespaceLabelKey, networkName))

	if networkName == "default" {
		labelFilter.Add("name", "k2d_net")
	}

	network, err := adapter.cli.NetworkList(ctx, types.NetworkListOptions{Filters: labelFilter})
	if err != nil {
		return &types.NetworkResource{}, fmt.Errorf("unable to list networks: %w", err)
	}

	if len(network) == 0 {
		return nil, nil
	}

	return &network[0], nil
}
