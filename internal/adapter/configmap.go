package adapter

import (
	"context"
	"fmt"

	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/volume"
	k2dtypes "github.com/portainer/k2d/internal/adapter/types"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubernetes/pkg/apis/core"
)

func (adapter *KubeDockerAdapter) CreateConfigMap(ctx context.Context, configMap *corev1.ConfigMap) error {
	_, err := adapter.CreateVolume(ctx, VolumeOptions{
		VolumeName: configMap.Name,
		Labels: map[string]string{
			k2dtypes.GenericLabelKey: "configmap",
		},
	})
	if err != nil {
		return fmt.Errorf("unable to create volume: %w", err)
	}

	err = adapter.fileSystemStore.StoreConfigMap(configMap)
	if err != nil {
		return fmt.Errorf("unable to store configmap: %w", err)
	}

	return nil
}

func (adapter *KubeDockerAdapter) DeleteConfigMap(configMapName string) error {
	filter := filters.NewArgs()
	filter.Add("label", fmt.Sprintf("%s=%s", k2dtypes.GenericLabelKey, "configmap"))
	filter.Add("name", configMapName)

	volume, err := adapter.cli.VolumeList(context.Background(), volume.ListOptions{
		Filters: filter,
	})
	if err != nil {
		return fmt.Errorf("unable to get the requested volume: %w", err)
	}

	if len(volume.Volumes) != 0 {
		err = adapter.cli.VolumeRemove(context.Background(), volume.Volumes[0].Name, true)
		if err != nil {
			return fmt.Errorf("unable to remove volume: %w", err)
		}
	}

	return nil
}

func (adapter *KubeDockerAdapter) GetConfigMap(configMapName string) (*corev1.ConfigMap, error) {
	configMap, err := adapter.fileSystemStore.GetConfigMap(configMapName)
	if err != nil {
		return &corev1.ConfigMap{}, fmt.Errorf("unable to get configmap: %w", err)
	}

	versionedConfigMap := corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
	}

	err = adapter.ConvertK8SResource(configMap, &versionedConfigMap)
	if err != nil {
		return nil, fmt.Errorf("unable to convert internal object to versioned object: %w", err)
	}

	versionedConfigMap.ObjectMeta.Annotations["kubectl.kubernetes.io/last-applied-configuration"] = ""

	return &versionedConfigMap, nil
}

func (adapter *KubeDockerAdapter) ListConfigMaps() (core.ConfigMapList, error) {
	labelFilter := filters.NewArgs()
	labelFilter.Add("label", fmt.Sprintf("%s=%s", k2dtypes.GenericLabelKey, "configmap"))

	volume, err := adapter.cli.VolumeList(context.Background(), volume.ListOptions{
		Filters: labelFilter,
	})

	if err != nil {
		return core.ConfigMapList{}, fmt.Errorf("unable to list volumes: %w", err)
	}

	mountPoints := []string{}
	for _, v := range volume.Volumes {
		mountPoints = append(mountPoints, v.Mountpoint)
	}

	return adapter.fileSystemStore.GetConfigMaps(mountPoints)
}
