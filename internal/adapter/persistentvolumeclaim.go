package adapter

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/errdefs"
	adaptererr "github.com/portainer/k2d/internal/adapter/errors"
	k2dtypes "github.com/portainer/k2d/internal/adapter/types"
	"github.com/portainer/k2d/internal/k8s"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubernetes/pkg/apis/core"
)

func (adapter *KubeDockerAdapter) CreatePersistentVolumeClaim(ctx context.Context, persistentVolumeClaim *corev1.PersistentVolumeClaim) error {
	if persistentVolumeClaim.Labels["app.kubernetes.io/managed-by"] == "Helm" {
		persistentVolumeClaimData, err := json.Marshal(persistentVolumeClaim)
		if err != nil {
			return fmt.Errorf("unable to marshal deployment: %w", err)
		}
		persistentVolumeClaim.ObjectMeta.Annotations["kubectl.kubernetes.io/last-applied-configuration"] = string(persistentVolumeClaimData)
	}

	_, err := adapter.cli.VolumeCreate(ctx, volume.CreateOptions{
		Name:   buildPersistentVolumeName(persistentVolumeClaim.Name, persistentVolumeClaim.Namespace),
		Driver: "local",
		Labels: map[string]string{
			k2dtypes.NamespaceLabelKey:                              persistentVolumeClaim.Namespace,
			k2dtypes.PersistentVolumeLabelKey:                       buildPersistentVolumeName(persistentVolumeClaim.Name, persistentVolumeClaim.Namespace),
			k2dtypes.PersistentVolumeClaimLabelKey:                  persistentVolumeClaim.Name,
			k2dtypes.PersistentVolumeClaimLastAppliedConfigLabelKey: persistentVolumeClaim.ObjectMeta.Annotations["kubectl.kubernetes.io/last-applied-configuration"],
		},
	})

	if err != nil {
		return fmt.Errorf("unable to create a Docker volume for the request persistent volume claim: %w", err)
	}

	return nil
}

func (adapter *KubeDockerAdapter) DeletePersistentVolumeClaim(ctx context.Context, persistentVolumeClaimName string, namespaceName string) error {
	persistentVolumeName := buildPersistentVolumeName(persistentVolumeClaimName, namespaceName)

	err := adapter.cli.VolumeRemove(ctx, persistentVolumeName, true)
	if err != nil {
		return fmt.Errorf("unable to remove Docker volume: %w", err)
	}

	return nil
}

func (adapter *KubeDockerAdapter) GetPersistentVolumeClaim(ctx context.Context, persistentVolumeClaimName string, namespaceName string) (*corev1.PersistentVolumeClaim, error) {
	volumeName := buildPersistentVolumeName(persistentVolumeClaimName, namespaceName)
	volume, err := adapter.cli.VolumeInspect(ctx, volumeName)
	if err != nil {
		if errdefs.IsNotFound(err) {
			return nil, adaptererr.ErrResourceNotFound
		}
		return nil, fmt.Errorf("unable to inspect docker volume %s: %w", volumeName, err)
	}

	persistentVolumeClaim, err := adapter.updatePersistentVolumeClaimFromVolume(volume)
	if err != nil {
		return nil, fmt.Errorf("unable to update persistent volume claim from volume: %w", err)
	}

	versionedpersistentVolumeClaim := corev1.PersistentVolumeClaim{
		TypeMeta: metav1.TypeMeta{
			Kind:       "PersistentVolumeClaim",
			APIVersion: "v1",
		},
	}

	err = adapter.ConvertK8SResource(persistentVolumeClaim, &versionedpersistentVolumeClaim)
	if err != nil {
		return nil, fmt.Errorf("unable to convert internal object to versioned object: %w", err)
	}

	return &versionedpersistentVolumeClaim, nil
}

func (adapter *KubeDockerAdapter) updatePersistentVolumeClaimFromVolume(volume volume.Volume) (*core.PersistentVolumeClaim, error) {
	persistentVolumeClaimData := volume.Labels[k2dtypes.PersistentVolumeClaimLastAppliedConfigLabelKey]

	versionedPersistentVolumeClaim := &corev1.PersistentVolumeClaim{}

	err := json.Unmarshal([]byte(persistentVolumeClaimData), &versionedPersistentVolumeClaim)
	if err != nil {
		return nil, fmt.Errorf("unable to unmarshal versioned service: %w", err)
	}

	persistentVolumeClaim := core.PersistentVolumeClaim{}
	err = adapter.ConvertK8SResource(versionedPersistentVolumeClaim, &persistentVolumeClaim)
	if err != nil {
		return nil, fmt.Errorf("unable to convert internal object to versioned object: %w", err)
	}

	err = adapter.converter.UpdateVolumeToPersistentVolumeClaim(&persistentVolumeClaim, volume)
	if err != nil {
		return nil, fmt.Errorf("unable to convert Docker volume to PersistentVolumeClaim: %w", err)
	}

	return &persistentVolumeClaim, nil
}

func (adapter *KubeDockerAdapter) ListPersistentVolumeClaims(ctx context.Context, namespaceName string) (core.PersistentVolumeClaimList, error) {
	persistentVolumeClaims, err := adapter.listPersistentVolumeClaims(ctx, namespaceName)
	if err != nil {
		return core.PersistentVolumeClaimList{}, fmt.Errorf("unable to list persistent volume claims: %w", err)
	}

	return *persistentVolumeClaims, nil
}

func (adapter *KubeDockerAdapter) GetPersistentVolumeClaimTable(ctx context.Context, namespaceName string) (*metav1.Table, error) {
	persistentVolumeClaims, err := adapter.listPersistentVolumeClaims(ctx, namespaceName)
	if err != nil {
		return &metav1.Table{}, fmt.Errorf("unable to list nodes: %w", err)
	}

	return k8s.GenerateTable(persistentVolumeClaims)
}

func (adapter *KubeDockerAdapter) listPersistentVolumeClaims(ctx context.Context, namespaceName string) (*core.PersistentVolumeClaimList, error) {
	labelFilter := filters.NewArgs()
	labelFilter.Add("label", k2dtypes.PersistentVolumeClaimLabelKey)
	labelFilter.Add("label", fmt.Sprintf("%s=%s", k2dtypes.NamespaceLabelKey, namespaceName))

	volumes, err := adapter.cli.VolumeList(ctx, volume.ListOptions{Filters: labelFilter})
	if err != nil {
		return &core.PersistentVolumeClaimList{}, fmt.Errorf("unable to list volumes to return the output values from a Docker volume: %w", err)
	}

	persistentVolumeClaims := core.PersistentVolumeClaimList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "PersistentVolumeClaimList",
			APIVersion: "v1",
		},
	}

	for _, volume := range volumes.Volumes {
		persistentVolumeClaim, err := adapter.updatePersistentVolumeClaimFromVolume(*volume)
		if err != nil {
			return nil, fmt.Errorf("unable to update persistent volume claim from volume: %w", err)
		}

		persistentVolumeClaims.Items = append(persistentVolumeClaims.Items, *persistentVolumeClaim)
	}

	return &persistentVolumeClaims, nil
}
