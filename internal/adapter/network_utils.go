package adapter

import (
	"context"
	"fmt"

	"github.com/docker/docker/api/types"
	k2dtypes "github.com/portainer/k2d/internal/adapter/types"
)

// findMatchingNetwork searches through the provided slice of NetworkResource types,
// looking for a network resource that matches the provided networkName.
// If a matching network resource is found, it returns a pointer to it.
// If no match is found, it returns nil.
func findMatchingNetwork(networks []types.NetworkResource, networkName string) *types.NetworkResource {
	for _, network := range networks {
		if network.Name == networkName {
			return &network
		}
	}

	return nil
}

// EnsureRequiredDockerResourcesExist verifies the existence of required Docker resources
// for k2d to work. If the required resources do not exist, it will attempt to create them.
//
// Specifically, this function checks for the existence of a Docker network with a specific name
// (K2DNetworkName). If this network does not exist, the function attempts to create it.
//
// It returns an error if it fails to list Docker networks or if it fails to create the required Docker network.
func (adapter *KubeDockerAdapter) EnsureRequiredDockerResourcesExist(ctx context.Context) error {
	networks, err := adapter.cli.NetworkList(ctx, types.NetworkListOptions{})
	if err != nil {
		return fmt.Errorf("unable to list docker networks: %w", err)
	}

	k2dNetwork := findMatchingNetwork(networks, k2dtypes.K2DNetworkName)
	if k2dNetwork == nil {
		adapter.logger.Info("creating k2d container network")
		_, err := adapter.cli.NetworkCreate(ctx, k2dtypes.K2DNetworkName, types.NetworkCreate{})
		if err != nil {
			return fmt.Errorf("unable to create k2d container network: %w", err)
		}

	} else {
		adapter.logger.Info("k2d container network already exists, skipping creation")
	}

	return nil
}
