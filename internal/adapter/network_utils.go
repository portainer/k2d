package adapter

import (
	"context"
	"errors"
	"fmt"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/errdefs"
	k2dtypes "github.com/portainer/k2d/internal/adapter/types"
)

// TODO: potentially need an errors package in the adapter package
var ErrNetworkNotFound = errors.New("network not found")

// TODO: update comment
// EnsureRequiredDockerResourcesExist verifies the existence of required Docker resources
// for k2d to work. If the required resources do not exist, it will attempt to create them.
//
// Specifically, this function checks for the existence of a Docker network with a specific name
// (K2DNetworkName). If this network does not exist, the function attempts to create it.
//
// It returns an error if it fails to list Docker networks or if it fails to create the required Docker network.
func (adapter *KubeDockerAdapter) EnsureRequiredDockerResourcesExist(ctx context.Context) error {
	k2dNetwork, err := adapter.GetNetwork(ctx, "default")
	if err != nil && !errors.Is(err, ErrNetworkNotFound) {
		return fmt.Errorf("unable to retrieve network: %w", err)
	}

	if k2dNetwork != nil {
		adapter.logger.Info("k2d container network already exists, skipping creation")
		return nil
	}

	adapter.logger.Info("creating k2d container network")

	_, err = adapter.cli.NetworkCreate(ctx, k2dtypes.K2DNetworkName, types.NetworkCreate{
		Labels: map[string]string{
			k2dtypes.NamespaceLabelKey: "default",
		},
	})
	if err != nil {
		return fmt.Errorf("unable to create k2d container network: %w", err)
	}

	return nil
}

// TODO: once k2d- prefixed networks are introduced this should not be needed anymore
// Special handling will be required for not found networks
func (adapter *KubeDockerAdapter) GetNetwork(ctx context.Context, networkName string) (*types.NetworkResource, error) {
	if networkName == "default" {
		networkName = k2dtypes.K2DNetworkName
	}

	network, err := adapter.cli.NetworkInspect(ctx, networkName, types.NetworkInspectOptions{})
	if err != nil {
		if errdefs.IsNotFound(err) {
			return nil, ErrNetworkNotFound
		}
		return nil, fmt.Errorf("unable to inspect network %s: %w", networkName, err)
	}

	return &network, nil
}
