package adapter

import (
	"context"
	"fmt"

	"github.com/portainer/k2d/internal/adapter/converter"
	adaptererr "github.com/portainer/k2d/internal/adapter/errors"
	"github.com/portainer/k2d/internal/k8s"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubernetes/pkg/apis/storage"
)

func (adapter *KubeDockerAdapter) GetStorageClass(ctx context.Context, storageClassName string) (*storagev1.StorageClass, error) {
	if storageClassName != "local" {
		return nil, adaptererr.ErrResourceNotFound
	}

	defaultStorageClass := converter.BuildDefaultStorageClass(adapter.startTime)

	versionedStorageClass := storagev1.StorageClass{
		TypeMeta: metav1.TypeMeta{
			Kind:       "StorageClass",
			APIVersion: "storage.k8s.io/v1",
		},
	}

	err := adapter.ConvertK8SResource(&defaultStorageClass, &versionedStorageClass)
	if err != nil {
		return nil, fmt.Errorf("unable to convert internal object to versioned object: %w", err)
	}

	return &versionedStorageClass, nil
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
	defaultStorageClass := converter.BuildDefaultStorageClass(adapter.startTime)

	storageClasses := []storage.StorageClass{}
	storageClasses = append(storageClasses, defaultStorageClass)

	return storage.StorageClassList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "StorageClassList",
			APIVersion: "storage.k8s.io/v1",
		},
		Items: storageClasses,
	}, nil
}
