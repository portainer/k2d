package adapter

import (
	"context"
	"fmt"

	"github.com/docker/docker/api/types"
)

func (adapter *KubeDockerAdapter) Ping(ctx context.Context) (types.Ping, error) {
	return adapter.cli.Ping(ctx)
}

func (adapter *KubeDockerAdapter) InfoAndVersion(ctx context.Context) (types.Info, types.Version, error) {
	info, err := adapter.cli.Info(ctx)
	if err != nil {
		return types.Info{}, types.Version{}, fmt.Errorf("unable to retrieve Docker info: %w", err)
	}

	version, err := adapter.cli.ServerVersion(ctx)
	if err != nil {
		return types.Info{}, types.Version{}, fmt.Errorf("unable to retrieve Docker version: %w", err)
	}

	return info, version, nil
}
