package converter

import (
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubernetes/pkg/apis/core"
	"k8s.io/kubernetes/pkg/apis/storage"
)

func BuildDefaultStorageClass(startTime time.Time) storage.StorageClass {
	reclaimPolicy := core.PersistentVolumeReclaimRetain
	volumeBindingMode := storage.VolumeBindingWaitForFirstConsumer

	return storage.StorageClass{
		ObjectMeta: metav1.ObjectMeta{
			Name: "local",
			Annotations: map[string]string{
				"storageclass.kubernetes.io/is-default-class": "true",
			},
			CreationTimestamp: metav1.Time{
				Time: startTime,
			},
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "StorageClass",
			APIVersion: "storage.k8s.io/v1",
		},
		Provisioner:       "k2d.io/local",
		ReclaimPolicy:     &reclaimPolicy,
		VolumeBindingMode: &volumeBindingMode,
	}
}
