package converter

import (
	"fmt"
	"time"

	"github.com/portainer/k2d/internal/adapter/naming"
	k2dtypes "github.com/portainer/k2d/internal/adapter/types"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubernetes/pkg/apis/core"
)

func (converter *DockerAPIConverter) UpdateConfigMapToPersistentVolumeClaim(persistentVolumeClaim *core.PersistentVolumeClaim, configMap *corev1.ConfigMap) error {
	creationDate, err := time.Parse(time.RFC3339, configMap.Labels["store.k2d.io/filesystem/creation-timestamp"])
	if err != nil {
		return fmt.Errorf("unable to parse persistent volume claim creation date: %w", err)
	}

	storageClassName := "local"

	persistentVolumeClaim.TypeMeta = metav1.TypeMeta{
		Kind:       "PersistentVolumeClaim",
		APIVersion: "v1",
	}

	persistentVolumeClaim.ObjectMeta = metav1.ObjectMeta{
		Name:      configMap.Labels[k2dtypes.PersistentVolumeClaimLabelKey],
		Namespace: configMap.Labels[k2dtypes.NamespaceLabelKey],
		CreationTimestamp: metav1.Time{
			Time: creationDate,
		},
		Annotations: map[string]string{
			"kubectl.kubernetes.io/last-applied-configuration": configMap.Labels[k2dtypes.PersistentVolumeClaimLastAppliedConfigLabelKey],
		},
	}

	persistentVolumeClaim.Spec = core.PersistentVolumeClaimSpec{
		StorageClassName: &storageClassName,
		VolumeName:       naming.BuildPersistentVolumeName(configMap.Labels[k2dtypes.PersistentVolumeClaimLabelKey], configMap.Labels[k2dtypes.NamespaceLabelKey]),
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
