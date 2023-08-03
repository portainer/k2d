package adapter

import (
	"context"
	"fmt"

	"github.com/docker/docker/api/types/volume"
)

type VolumeOptions struct {
	VolumeName string
	Labels     map[string]string
}

func (adapter *KubeDockerAdapter) CreateVolume(ctx context.Context, options VolumeOptions) (volume.Volume, error) {
	out, err := adapter.cli.VolumeCreate(ctx, volume.CreateOptions{
		Name:   options.VolumeName,
		Labels: options.Labels,
	})

	if err != nil {
		return volume.Volume{}, fmt.Errorf("unable to create the %s docker volume: %w", options.VolumeName, err)
	}

	return out, nil
}

func (adapter *KubeDockerAdapter) ListVolume(ctx context.Context, options VolumeOptions) (volume.Volume, error) {
	out, err := adapter.cli.VolumeCreate(ctx, volume.CreateOptions{
		Name:   options.VolumeName,
		Labels: options.Labels,
	})

	if err != nil {
		return volume.Volume{}, fmt.Errorf("unable to create the %s docker volume: %w", options.VolumeName, err)
	}

	return out, nil
}

func (adapter *KubeDockerAdapter) GetVolume(ctx context.Context, options VolumeOptions) (volume.Volume, error) {
	out, err := adapter.cli.VolumeCreate(ctx, volume.CreateOptions{
		Name:   options.VolumeName,
		Labels: options.Labels,
	})

	if err != nil {
		return volume.Volume{}, fmt.Errorf("unable to create the %s docker volume: %w", options.VolumeName, err)
	}

	return out, nil
}

func (adapter *KubeDockerAdapter) DeleteVolume(ctx context.Context, options VolumeOptions) (volume.Volume, error) {
	out, err := adapter.cli.VolumeCreate(ctx, volume.CreateOptions{
		Name:   options.VolumeName,
		Labels: options.Labels,
	})

	if err != nil {
		return volume.Volume{}, fmt.Errorf("unable to create the %s docker volume: %w", options.VolumeName, err)
	}

	return out, nil
}
