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
			AccessModes: []core.PersistentVolumeAccessMode{
				core.ReadWriteOnce,
			},
			PersistentVolumeReclaimPolicy: core.PersistentVolumeReclaimRetain,
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
