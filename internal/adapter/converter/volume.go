package converter

import (
	"fmt"
	"time"

	"github.com/docker/docker/api/types/volume"
	k2dtypes "github.com/portainer/k2d/internal/adapter/types"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (converter *DockerAPIConverter) ConvertVolumeToPersistentVolume(volume volume.Volume) (*corev1.PersistentVolume, error) {
	creationDate, err := time.Parse(time.RFC3339, volume.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("unable to parse volume creation date: %w", err)
	}

	resourceList := corev1.ResourceList{}
	if volume.UsageData != nil {
		resourceList[corev1.ResourceStorage] = resource.MustParse(fmt.Sprint(volume.UsageData.Size))
	}

	// claim reference
	persistentVolumeClaimReference := &corev1.ObjectReference{
		Kind:      "PersistentVolumeClaim",
		Namespace: volume.Labels[k2dtypes.NamespaceLabelKey],
		Name:      volume.Labels[k2dtypes.PersistentVolumeClaimLabelKey],
	}

	return &corev1.PersistentVolume{
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
		Spec: corev1.PersistentVolumeSpec{
			Capacity: resourceList,
			AccessModes: []corev1.PersistentVolumeAccessMode{
				corev1.ReadWriteOnce,
			},
			PersistentVolumeReclaimPolicy: corev1.PersistentVolumeReclaimDelete,
			PersistentVolumeSource: corev1.PersistentVolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: volume.Mountpoint,
				},
			},
			ClaimRef: persistentVolumeClaimReference,
		},
		Status: corev1.PersistentVolumeStatus{
			Phase: corev1.VolumeBound,
		},
	}, nil
}
