package adapter

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/volume"
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
		// DriverOpts: map[string]string{
		// 	"o": "size" + persistentVolumeClaim.Spec.Resources.Requests.Storage().String(),
		// },
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
	labelFilter := filters.NewArgs()
	labelFilter.Add("name", persistentVolumeClaimName)
	labelFilter.Add("label", fmt.Sprintf("%s=%s", k2dtypes.NamespaceLabelKey, namespaceName))

	volumes, err := adapter.cli.VolumeList(ctx, volume.ListOptions{Filters: labelFilter})
	if err != nil {
		return nil, fmt.Errorf("unable to list volumes to return the output values from a Docker volume: %w", err)
	}

	if len(volumes.Volumes) == 0 {
		return nil, adaptererr.ErrResourceNotFound
	}

	if len(volumes.Volumes) > 1 {
		return nil, fmt.Errorf("multiple volumes were found with the associated persistent volume claim name %s", persistentVolumeClaimName)
	}

	persistentVolumeClaim, err := adapter.buildPersistentVolumeClaimFromVolume(*volumes.Volumes[0])
	if err != nil {
		return nil, fmt.Errorf("unable to build persistent volume claim from volume: %w", err)
	}

	return persistentVolumeClaim, nil
}

func (adapter *KubeDockerAdapter) buildPersistentVolumeClaimFromVolume(volume volume.Volume) (*corev1.PersistentVolumeClaim, error) {
	if volume.Labels[k2dtypes.PersistentVolumeClaimLastAppliedConfigLabelKey] == "" {
		return nil, fmt.Errorf("unable to build deployment, missing %s label on container %s", k2dtypes.PersistentVolumeClaimLastAppliedConfigLabelKey, volume.Name)
	}

	persistentVolumeClaimData := volume.Labels[k2dtypes.PersistentVolumeClaimLastAppliedConfigLabelKey]

	versionedPersistentVolumeClaim := corev1.PersistentVolumeClaim{}

	err := json.Unmarshal([]byte(persistentVolumeClaimData), &versionedPersistentVolumeClaim)
	if err != nil {
		return nil, fmt.Errorf("unable to unmarshal persistent volume claim: %w", err)
	}

	persistentVolumeClaim := corev1.PersistentVolumeClaim{}
	err = adapter.ConvertK8SResource(&versionedPersistentVolumeClaim, &persistentVolumeClaim)
	if err != nil {
		return nil, fmt.Errorf("unable to convert versioned deployment spec to internal persistent volume claim spec: %w", err)
	}

	creationDate, err := time.Parse(time.RFC3339, volume.CreatedAt)
	if err != nil {
		return &corev1.PersistentVolumeClaim{}, fmt.Errorf("unable to parse volume creation date: %w", err)
	}

	storageClassName := "local"

	return &corev1.PersistentVolumeClaim{
		TypeMeta: metav1.TypeMeta{
			Kind:       "PersistentVolumeClaim",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      volume.Labels[k2dtypes.PersistentVolumeClaimLabelKey],
			Namespace: volume.Labels[k2dtypes.NamespaceLabelKey],
			CreationTimestamp: metav1.Time{
				Time: creationDate,
			},
			Annotations: map[string]string{
				"kubectl.kubernetes.io/last-applied-configuration": volume.Labels[k2dtypes.PersistentVolumeClaimLastAppliedConfigLabelKey],
			},
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			StorageClassName: &storageClassName,
			VolumeName:       buildPersistentVolumeName(volume.Labels[k2dtypes.PersistentVolumeClaimLabelKey], volume.Labels[k2dtypes.NamespaceLabelKey]),
			AccessModes: []corev1.PersistentVolumeAccessMode{
				corev1.ReadWriteOnce,
			},
		},
		Status: corev1.PersistentVolumeClaimStatus{
			Phase: corev1.ClaimBound,
			AccessModes: []corev1.PersistentVolumeAccessMode{
				corev1.ReadWriteOnce,
			},
		},
	}, nil
}

func (adapter *KubeDockerAdapter) ListPersistentVolumeClaims(ctx context.Context, namespaceName string) (core.PersistentVolumeClaimList, error) {
	persistentVolumeClaims, err := adapter.listPersistentVolumeClaims(ctx, namespaceName)
	if err != nil {
		return core.PersistentVolumeClaimList{}, fmt.Errorf("unable to list persistent volume claims: %w", err)
	}

	return persistentVolumeClaims, nil
}

func (adapter *KubeDockerAdapter) GetPersistentVolumeClaimTable(ctx context.Context, namespaceName string) (*metav1.Table, error) {
	persistentVolumeClaims, err := adapter.listPersistentVolumeClaims(ctx, namespaceName)
	if err != nil {
		return &metav1.Table{}, fmt.Errorf("unable to list nodes: %w", err)
	}

	return k8s.GenerateTable(&persistentVolumeClaims)
}

func (adapter *KubeDockerAdapter) listPersistentVolumeClaims(ctx context.Context, namespaceName string) (core.PersistentVolumeClaimList, error) {
	labelFilter := filters.NewArgs()
	labelFilter.Add("label", k2dtypes.PersistentVolumeClaimLabelKey)
	labelFilter.Add("label", fmt.Sprintf("%s=%s", k2dtypes.NamespaceLabelKey, namespaceName))

	volumeList, err := adapter.cli.VolumeList(ctx, volume.ListOptions{Filters: labelFilter})
	if err != nil {
		return core.PersistentVolumeClaimList{}, fmt.Errorf("unable to list volumes to return the output values from a Docker volume: %w", err)
	}

	persistentVolumeClaims := core.PersistentVolumeClaimList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "PersistentVolumeClaimList",
			APIVersion: "v1",
		},
	}

	// TODO: storage class name determination

	for _, volume := range volumeList.Volumes {
		creationDate, err := time.Parse(time.RFC3339, volume.CreatedAt)
		if err != nil {
			return core.PersistentVolumeClaimList{}, fmt.Errorf("unable to parse volume creation date: %w", err)
		}

		storageClassName := "local"

		persistentVolumeClaims.Items = append(persistentVolumeClaims.Items, core.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name:      volume.Labels[k2dtypes.PersistentVolumeClaimLabelKey],
				Namespace: namespaceName,
				CreationTimestamp: metav1.Time{
					Time: creationDate,
				},
			},
			TypeMeta: metav1.TypeMeta{
				Kind:       "PersistentVolumeClaim",
				APIVersion: "v1",
			},
			Spec: core.PersistentVolumeClaimSpec{
				StorageClassName: &storageClassName,
				VolumeName:       buildPersistentVolumeName(volume.Labels[k2dtypes.PersistentVolumeClaimLabelKey], namespaceName),
				AccessModes: []core.PersistentVolumeAccessMode{
					core.ReadWriteOnce,
				},
			},
			Status: core.PersistentVolumeClaimStatus{
				Phase: core.ClaimBound,
				AccessModes: []core.PersistentVolumeAccessMode{
					core.ReadWriteOnce,
				},
			},
		})
	}

	return persistentVolumeClaims, nil
}
