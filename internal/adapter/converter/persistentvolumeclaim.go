package converter

import (
	k2dtypes "github.com/portainer/k2d/internal/adapter/types"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubernetes/pkg/apis/core"
)

func (converter *DockerAPIConverter) UpdateConfigMapToPersistentVolumeClaim(persistentVolumeClaim *core.PersistentVolumeClaim, configMap *corev1.ConfigMap) error {
	storageClassName := "local"

	persistentVolumeClaim.TypeMeta = metav1.TypeMeta{
		Kind:       "PersistentVolumeClaim",
		APIVersion: "v1",
	}

	persistentVolumeClaim.ObjectMeta = metav1.ObjectMeta{
		Name:      configMap.Labels[k2dtypes.PersistentVolumeClaimNameLabelKey],
		Namespace: configMap.Labels[k2dtypes.PersistentVolumeClaimTargetNamespaceLabelKey],
		CreationTimestamp: metav1.Time{
			Time: configMap.CreationTimestamp.Time,
		},
		Annotations: map[string]string{
			"kubectl.kubernetes.io/last-applied-configuration": configMap.Labels[k2dtypes.LastAppliedConfigLabelKey],
		},
	}

	persistentVolumeClaim.Spec = core.PersistentVolumeClaimSpec{
		StorageClassName: &storageClassName,
		VolumeName:       configMap.Labels[k2dtypes.PersistentVolumeNameLabelKey],
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
