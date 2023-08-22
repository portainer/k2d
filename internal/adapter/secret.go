package adapter

import (
	"context"
	"fmt"

	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/volume"
	k2dtypes "github.com/portainer/k2d/internal/adapter/types"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/kubernetes/pkg/apis/core"
)

func (adapter *KubeDockerAdapter) CreateSecret(ctx context.Context, secret *corev1.Secret) error {
	// return adapter.fileSystemStore.StoreSecret(secret)
	_, err := adapter.CreateVolume(ctx, VolumeOptions{
		VolumeName: secret.Name,
		Labels: map[string]string{
			k2dtypes.GenericLabelKey: "secret",
		},
	})
	if err != nil {
		return fmt.Errorf("unable to create volume: %w", err)
	}

	err = adapter.fileSystemStore.StoreSecret(secret)
	if err != nil {
		return fmt.Errorf("unable to store secret: %w", err)
	}

	return nil
}

func (adapter *KubeDockerAdapter) DeleteSecret(secretName string) error {
	filter := filters.NewArgs()
	filter.Add("label", fmt.Sprintf("%s=%s", k2dtypes.GenericLabelKey, "secret"))
	filter.Add("name", secretName)

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

func (adapter *KubeDockerAdapter) GetSecret(secretName string) (*corev1.Secret, error) {
	secret, err := adapter.fileSystemStore.GetSecret(secretName)
	if err != nil {
		return &corev1.Secret{}, fmt.Errorf("unable to get secret: %w", err)
	}

	versionedSecret := corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
	}

	err = adapter.ConvertK8SResource(secret, &versionedSecret)
	if err != nil {
		return nil, fmt.Errorf("unable to convert internal object to versioned object: %w", err)
	}

	versionedSecret.ObjectMeta.Annotations["kubectl.kubernetes.io/last-applied-configuration"] = ""

	return &versionedSecret, nil
}

func (adapter *KubeDockerAdapter) ListSecrets() (core.SecretList, error) {
	labelFilter := filters.NewArgs()
	labelFilter.Add("label", fmt.Sprintf("%s=%s", k2dtypes.GenericLabelKey, "secret"))

	volume, err := adapter.cli.VolumeList(context.Background(), volume.ListOptions{
		Filters: labelFilter,
	})

	if err != nil {
		return core.SecretList{}, fmt.Errorf("unable to list volumes: %w", err)
	}

	mountPoints := []string{}
	for _, v := range volume.Volumes {
		mountPoints = append(mountPoints, v.Mountpoint)
	}

	return adapter.fileSystemStore.GetSecrets(mountPoints)
}
