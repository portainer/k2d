package adapter

import (
	"context"
	"fmt"

	"github.com/portainer/k2d/internal/k8s"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubernetes/pkg/apis/storage"
)

func (adapter *KubeDockerAdapter) GetStorageClass(ctx context.Context, storageClassName string) (*storagev1.StorageClass, error) {
	info, _, err := adapter.InfoAndVersion(ctx)
	if err != nil {
		return &storagev1.StorageClass{}, err
	}

	for _, plugin := range info.Plugins.Volume {
		if plugin == storageClassName {
			return &storagev1.StorageClass{
				ObjectMeta: metav1.ObjectMeta{
					Name: plugin,
				},
				TypeMeta: metav1.TypeMeta{
					Kind:       "StorageClass",
					APIVersion: "storage.k8s.io/v1",
				},
				Provisioner: plugin,
			}, nil
		}
	}

	return nil, nil
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
	info, _, err := adapter.InfoAndVersion(ctx)
	if err != nil {
		return storage.StorageClassList{}, err
	}

	storageClasses := []storage.StorageClass{}

	for _, plugin := range info.Plugins.Volume {
		storageClasses = append(storageClasses, storage.StorageClass{
			ObjectMeta: metav1.ObjectMeta{
				Name: plugin,
			},
			TypeMeta: metav1.TypeMeta{
				Kind:       "StorageClass",
				APIVersion: "storage.k8s.io/v1",
			},
			Provisioner: plugin,
		})
	}

	return storage.StorageClassList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "StorageClassList",
			APIVersion: "storage.k8s.io/v1",
		},
		Items: storageClasses,
	}, nil
}
