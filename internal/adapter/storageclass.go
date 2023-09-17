package adapter

import (
	"context"
	"fmt"

	adaptererr "github.com/portainer/k2d/internal/adapter/errors"
	"github.com/portainer/k2d/internal/k8s"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubernetes/pkg/apis/core"
	"k8s.io/kubernetes/pkg/apis/storage"
)

func (adapter *KubeDockerAdapter) GetStorageClass(ctx context.Context, storageClassName string) (*storagev1.StorageClass, error) {
	if storageClassName != "local" {
		return nil, adaptererr.ErrResourceNotFound
	}

	reclaimPolicy := corev1.PersistentVolumeReclaimPolicy("Retain")
	volumeBindingMode := storagev1.VolumeBindingMode("WaitForFirstConsumer")

	return &storagev1.StorageClass{
		ObjectMeta: metav1.ObjectMeta{
			Name: "local",
			Annotations: map[string]string{
				"storageclass.kubernetes.io/is-default-class": "true",
			},
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "StorageClass",
			APIVersion: "storage.k8s.io/v1",
		},
		Provisioner:       "k2d.io/local",
		ReclaimPolicy:     &reclaimPolicy,
		VolumeBindingMode: &volumeBindingMode,
	}, nil
}

func (adapter *KubeDockerAdapter) ListStorageClasses(ctx context.Context) (storagev1.StorageClassList, error) {
	storageClassList, err := adapter.listStorageClasses(ctx)
	if err != nil {
		return storagev1.StorageClassList{}, fmt.Errorf("unable to list storage classes: %w", err)
	}

	versionedStorageClassList := storagev1.StorageClassList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "StorageClassList",
			APIVersion: "storage.k8s.io/v1",
		},
	}

	err = adapter.ConvertK8SResource(&storageClassList, &versionedStorageClassList)
	if err != nil {
		return storagev1.StorageClassList{}, fmt.Errorf("unable to convert internal StorageClassList to versioned StorageClassList: %w", err)
	}

	return versionedStorageClassList, nil
}

func (adapter *KubeDockerAdapter) GetStorageClassTable(ctx context.Context) (*metav1.Table, error) {
	storageClassList, err := adapter.listStorageClasses(ctx)
	if err != nil {
		return &metav1.Table{}, fmt.Errorf("unable to list storageClasses: %w", err)
	}

	return k8s.GenerateTable(&storageClassList)
}

func (adapter *KubeDockerAdapter) listStorageClasses(ctx context.Context) (storage.StorageClassList, error) {
	storageClasses := []storage.StorageClass{}

	reclaimPolicy := core.PersistentVolumeReclaimPolicy("Retain")
	volumeBindingMode := storage.VolumeBindingMode("WaitForFirstConsumer")

	storageClasses = append(storageClasses, storage.StorageClass{
		ObjectMeta: metav1.ObjectMeta{
			Name: "local",
			Annotations: map[string]string{
				"storageclass.kubernetes.io/is-default-class": "true",
			},
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "StorageClass",
			APIVersion: "storage.k8s.io/v1",
		},
		Provisioner:       "k2d.io/local",
		ReclaimPolicy:     &reclaimPolicy,
		VolumeBindingMode: &volumeBindingMode,
	})

	return storage.StorageClassList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "StorageClassList",
			APIVersion: "storage.k8s.io/v1",
		},
		Items: storageClasses,
	}, nil
}
