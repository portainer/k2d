package converter

import (
	"fmt"
	"time"

	"github.com/docker/docker/api/types/volume"
	"github.com/portainer/k2d/internal/adapter/naming"
	k2dtypes "github.com/portainer/k2d/internal/adapter/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubernetes/pkg/apis/core"
)

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
		VolumeName:       naming.BuildPersistentVolumeName(volume.Labels[k2dtypes.PersistentVolumeClaimLabelKey], volume.Labels[k2dtypes.NamespaceLabelKey]),
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
