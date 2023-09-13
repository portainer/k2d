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

	return &storagev1.StorageClass{
		ObjectMeta: metav1.ObjectMeta{
			Name: "local",
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "StorageClass",
			APIVersion: "storage.k8s.io/v1",
		},
		Provisioner:   "local",
		ReclaimPolicy: &reclaimPolicy,
	}, nil
}

func (adapter *KubeDockerAdapter) ListStorageClasses(ctx context.Context) (storage.StorageClassList, error) {
	storageClassList, err := adapter.listStorageClasses(ctx)
	if err != nil {
		return storage.StorageClassList{}, fmt.Errorf("unable to list storageClasses: %w", err)
	}

	return storageClassList, nil
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
	storageClasses = append(storageClasses, storage.StorageClass{
		ObjectMeta: metav1.ObjectMeta{
			Name: "local",
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "StorageClass",
			APIVersion: "storage.k8s.io/v1",
		},
		Provisioner:   "local",
		ReclaimPolicy: &reclaimPolicy,
	})

	return storage.StorageClassList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "StorageClassList",
			APIVersion: "storage.k8s.io/v1",
		},
		Items: storageClasses,
	}, nil
}
