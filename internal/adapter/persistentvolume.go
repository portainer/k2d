package adapter

import (
	"context"
	"fmt"

	"github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/errdefs"
	adaptererr "github.com/portainer/k2d/internal/adapter/errors"
	"github.com/portainer/k2d/internal/adapter/filters"
	k2dtypes "github.com/portainer/k2d/internal/adapter/types"
	"github.com/portainer/k2d/internal/k8s"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubernetes/pkg/apis/core"
)

func (adapter *KubeDockerAdapter) DeletePersistentVolume(ctx context.Context, persistentVolumeName string) error {
	err := adapter.cli.VolumeRemove(ctx, persistentVolumeName, true)
	if err != nil {
		return fmt.Errorf("unable to remove Docker volume: %w", err)
	}

	return nil
}

func (adapter *KubeDockerAdapter) GetPersistentVolume(ctx context.Context, persistentVolumeName string) (*corev1.PersistentVolume, error) {
	volume, err := adapter.cli.VolumeInspect(ctx, persistentVolumeName)
	if err != nil {
		if errdefs.IsNotFound(err) {
			return nil, adaptererr.ErrResourceNotFound
		}
		return nil, fmt.Errorf("unable to inspect docker volume %s: %w", persistentVolumeName, err)
	}

	configMaps, err := adapter.ListSystemConfigMaps()
	if err != nil {
		return nil, fmt.Errorf("unable to list configmaps: %w", err)
	}

	var boundPVCConfigMap *corev1.ConfigMap
	for _, configMap := range configMaps.Items {
		if configMap.Labels[k2dtypes.PersistentVolumeNameLabelKey] == volume.Name {
			boundPVCConfigMap = &configMap
			break
		}
	}

	if boundPVCConfigMap == nil {
		adapter.logger.Debugw("unable to retrieve system configmap for volume, setting phase to released and no claim reference",
			"volume", volume.Name,
		)
	}

	persistentVolume, err := adapter.converter.ConvertVolumeToPersistentVolume(&volume, boundPVCConfigMap)
	if err != nil {
		return nil, fmt.Errorf("unable to convert Docker volume to PersistentVolume: %w", err)
	}

	versionedPersistentVolume := corev1.PersistentVolume{
		TypeMeta: metav1.TypeMeta{
			Kind:       "PersistentVolume",
			APIVersion: "v1",
		},
	}

	err = adapter.ConvertK8SResource(&persistentVolume, &versionedPersistentVolume)
	if err != nil {
		return nil, fmt.Errorf("unable to convert internal object to versioned object: %w", err)
	}

	return &versionedPersistentVolume, nil
}

func (adapter *KubeDockerAdapter) ListPersistentVolumes(ctx context.Context) (corev1.PersistentVolumeList, error) {
	persistentVolumes, err := adapter.listPersistentVolumes(ctx)
	if err != nil {
		return corev1.PersistentVolumeList{}, fmt.Errorf("unable to list nodes: %w", err)
	}

	versionedPersistentVolumeList := corev1.PersistentVolumeList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "PersistentVolumeList",
			APIVersion: "v1",
		},
	}

	err = adapter.ConvertK8SResource(&persistentVolumes, &versionedPersistentVolumeList)
	if err != nil {
		return corev1.PersistentVolumeList{}, fmt.Errorf("unable to convert internal PersistentVolumeList to versioned PersistentVolumeList: %w", err)
	}

	return versionedPersistentVolumeList, nil
}

func (adapter *KubeDockerAdapter) GetPersistentVolumeTable(ctx context.Context) (*metav1.Table, error) {
	persistentVolumeList, err := adapter.listPersistentVolumes(ctx)
	if err != nil {
		return &metav1.Table{}, fmt.Errorf("unable to list persistent volumes: %w", err)
	}

	return k8s.GenerateTable(&persistentVolumeList)
}

func (adapter *KubeDockerAdapter) listPersistentVolumes(ctx context.Context) (core.PersistentVolumeList, error) {
	filter := filters.AllPersistentVolumes()

	volumeList, err := adapter.cli.VolumeList(ctx, volume.ListOptions{Filters: filter})
	if err != nil {
		return core.PersistentVolumeList{}, fmt.Errorf("unable to list volumes to return the output values from a Docker volume: %w", err)
	}

	configMaps, err := adapter.ListSystemConfigMaps()
	if err != nil {
		return core.PersistentVolumeList{}, fmt.Errorf("unable to list system configmaps: %w", err)
	}

	persistentVolumes := []core.PersistentVolume{}

	for _, volume := range volumeList.Volumes {
		var boundPVCConfigMap *corev1.ConfigMap

		for _, configMap := range configMaps.Items {
			if configMap.Labels[k2dtypes.PersistentVolumeNameLabelKey] == volume.Name {
				boundPVCConfigMap = &configMap
				break
			}
		}

		if boundPVCConfigMap == nil {
			adapter.logger.Debugw("unable to retrieve system configmap for volume, setting phase to released and no claim reference",
				"volume", volume.Name,
			)
		}

		persistentVolume, err := adapter.converter.ConvertVolumeToPersistentVolume(volume, boundPVCConfigMap)
		if err != nil {
			return core.PersistentVolumeList{}, fmt.Errorf("unable to convert Docker volume to PersistentVolume: %w", err)
		}

		persistentVolumes = append(persistentVolumes, persistentVolume)

	}

	return core.PersistentVolumeList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "PersistentVolumeList",
			APIVersion: "v1",
		},
		Items: persistentVolumes,
	}, nil
}
