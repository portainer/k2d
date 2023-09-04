package converter

import (
	"fmt"
	"time"

	"github.com/docker/docker/api/types/volume"
	k2dtypes "github.com/portainer/k2d/internal/adapter/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubernetes/pkg/apis/core"
)

func (converter *DockerAPIConverter) ConvertVolumeToPersistentVolume(volume volume.Volume) (*core.PersistentVolume, error) {
	creationDate, err := time.Parse(time.RFC3339, volume.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("unable to parse volume creation date: %w", err)
	}

	persistentVolumeClaimReference := &core.ObjectReference{
		Kind:      "PersistentVolumeClaim",
		Namespace: volume.Labels[k2dtypes.NamespaceLabelKey],
		Name:      volume.Labels[k2dtypes.PersistentVolumeClaimLabelKey],
	}

	return &core.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name: volume.Name,
			CreationTimestamp: metav1.Time{
				Time: creationDate,
			},
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "PersistentVolume",
			APIVersion: "v1",
		},
		Spec: core.PersistentVolumeSpec{
			// Capacity: resourceList,
			AccessModes: []core.PersistentVolumeAccessMode{
				core.ReadWriteOnce,
			},
			PersistentVolumeReclaimPolicy: core.PersistentVolumeReclaimDelete,
			PersistentVolumeSource: core.PersistentVolumeSource{
				HostPath: &core.HostPathVolumeSource{
					Path: volume.Mountpoint,
				},
			},
			ClaimRef:         persistentVolumeClaimReference,
			StorageClassName: "local",
		},
		Status: core.PersistentVolumeStatus{
			Phase: core.VolumeBound,
		},
	}, nil
}

func (converter *DockerAPIConverter) UpdateVolumeToPersistentVolumeClaim(persistentVolumeClaim *core.PersistentVolumeClaim, volume volume.Volume) error {
	creationDate, err := time.Parse(time.RFC3339, volume.CreatedAt)
	if err != nil {
		return fmt.Errorf("unable to parse volume creation date: %w", err)
	}

	storageClassName := "local"

	persistentVolumeClaim.TypeMeta = metav1.TypeMeta{
		Kind:       "PersistentVolumeClaim",
		APIVersion: "v1",
	}

	persistentVolumeClaim.ObjectMeta = metav1.ObjectMeta{
		Name:      volume.Labels[k2dtypes.PersistentVolumeClaimLabelKey],
		Namespace: volume.Labels[k2dtypes.NamespaceLabelKey],
		CreationTimestamp: metav1.Time{
			Time: creationDate,
		},
		Annotations: map[string]string{
			"kubectl.kubernetes.io/last-applied-configuration": volume.Labels[k2dtypes.PersistentVolumeClaimLastAppliedConfigLabelKey],
		},
	}

	persistentVolumeClaim.Spec = core.PersistentVolumeClaimSpec{
		StorageClassName: &storageClassName,
		// TODO: Replace the fmt.Sprintf("k2d-pv-%s-%s", namespace, volume.VolumeSource.PersistentVolumeClaim.ClaimName)
		// part with the buildPersistentVolumeName function.
		VolumeName: fmt.Sprintf("k2d-pv-%s-%s", volume.Labels[k2dtypes.NamespaceLabelKey], volume.Labels[k2dtypes.PersistentVolumeClaimLabelKey]),
		AccessModes: []core.PersistentVolumeAccessMode{
			core.ReadWriteOnce,
		},
		Resources: persistentVolumeClaim.Spec.Resources,
	}

	persistentVolumeClaim.Status = core.PersistentVolumeClaimStatus{
		Phase: core.ClaimBound,
		AccessModes: []core.PersistentVolumeAccessMode{
			core.ReadWriteOnce,
		},
	}

	return nil
}
