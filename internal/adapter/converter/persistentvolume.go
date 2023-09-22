package converter

import (
	"fmt"
	"time"

	"github.com/docker/docker/api/types/volume"
	k2dtypes "github.com/portainer/k2d/internal/adapter/types"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubernetes/pkg/apis/core"
)

func (converter *DockerAPIConverter) ConvertVolumeToPersistentVolume(volume *volume.Volume, pvcConfigMap *corev1.ConfigMap) (core.PersistentVolume, error) {
	creationDate, err := time.Parse(time.RFC3339, volume.CreatedAt)
	if err != nil {
		return core.PersistentVolume{}, fmt.Errorf("unable to parse volume creation date: %w", err)
	}

	var persistentVolumeClaimReference *core.ObjectReference
	phase := core.VolumeReleased

	if pvcConfigMap != nil {
		phase = core.VolumeBound
		persistentVolumeClaimReference = &core.ObjectReference{
			Kind:      "PersistentVolumeClaim",
			Namespace: pvcConfigMap.Labels[k2dtypes.PersistentVolumeClaimTargetNamespaceLabelKey],
			Name:      pvcConfigMap.Labels[k2dtypes.PersistentVolumeClaimNameLabelKey],
		}
	}

	return core.PersistentVolume{
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
			Phase: phase,
		},
	}, nil
}
