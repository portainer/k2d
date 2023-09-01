package adapter

import (
	"context"
	"fmt"
	"time"

	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/volume"
	k2dtypes "github.com/portainer/k2d/internal/adapter/types"
	"github.com/portainer/k2d/internal/k8s"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubernetes/pkg/apis/core"
)

func (adapter *KubeDockerAdapter) GetPersistentVolume(ctx context.Context, persistentVolumeName string) (*corev1.PersistentVolume, error) {
	volume, err := adapter.cli.VolumeInspect(ctx, persistentVolumeName)
	if err != nil {
		return nil, fmt.Errorf("unable to list volumes to return the output values from a Docker volume: %w", err)
	}

	return adapter.converter.ConvertVolumeToPersistentVolume(volume)
}

func (adapter *KubeDockerAdapter) ListPersistentVolumes(ctx context.Context) (core.PersistentVolumeList, error) {
	persistentVolumes, err := adapter.listPersistentVolumes(ctx)
	if err != nil {
		return core.PersistentVolumeList{}, fmt.Errorf("unable to list nodes: %w", err)
	}

	return persistentVolumes, nil
}

func (adapter *KubeDockerAdapter) GetPersistentVolumeTable(ctx context.Context) (*metav1.Table, error) {
	persistentVolumeList, err := adapter.listPersistentVolumes(ctx)
	if err != nil {
		return &metav1.Table{}, fmt.Errorf("unable to list nodes: %w", err)
	}

	return k8s.GenerateTable(&persistentVolumeList)
}

func (adapter *KubeDockerAdapter) listPersistentVolumes(ctx context.Context) (core.PersistentVolumeList, error) {
	labelFilter := filters.NewArgs()
	labelFilter.Add("label", k2dtypes.PersistentVolumeLabelKey)

	volumeList, err := adapter.cli.VolumeList(ctx, volume.ListOptions{Filters: labelFilter})
	if err != nil {
		return core.PersistentVolumeList{}, fmt.Errorf("unable to list volumes to return the output values from a Docker volume: %w", err)
	}

	persistentVolumes := []core.PersistentVolume{}

	for volume := range volumeList.Volumes {
		creationDate, err := time.Parse(time.RFC3339, volumeList.Volumes[volume].CreatedAt)
		if err != nil {
			return core.PersistentVolumeList{}, fmt.Errorf("unable to parse volume creation date: %w", err)
		}

		resourceList := core.ResourceList{}
		if volumeList.Volumes[volume].UsageData != nil {
			resourceList[core.ResourceStorage] = resource.MustParse(fmt.Sprint(volumeList.Volumes[volume].UsageData.Size))
		}

		persistentClaimReference := &core.ObjectReference{
			Kind:      "PersistentVolumeClaim",
			Namespace: volumeList.Volumes[volume].Labels[k2dtypes.NamespaceLabelKey],
			Name:      volumeList.Volumes[volume].Labels[k2dtypes.PersistentVolumeClaimLabelKey],
		}

		// TODO: status of the persistent volume has to be retrieved from the
		// containers that are using it

		// "Mounts": [
		// 		{
		// 				"Type": "bind",
		// 				"Source": "/var/run/docker.sock",
		// 				"Destination": "/var/run/docker.sock",
		// 				"Mode": "z",
		// 				"RW": true,
		// 				"Propagation": "rprivate"
		// 		},
		// 		{
		// 				"Type": "volume",
		// 				"Name": "portainer_data",
		// 				"Source": "/var/lib/docker/volumes/portainer_data/_data",
		// 				"Destination": "/data",
		// 				"Driver": "local",
		// 				"Mode": "z",
		// 				"RW": true,
		// 				"Propagation": ""
		// 		}
		// ]

		persistentVolumes = append(persistentVolumes, core.PersistentVolume{
			ObjectMeta: metav1.ObjectMeta{
				Name: volumeList.Volumes[volume].Name,
				CreationTimestamp: metav1.Time{
					Time: creationDate,
				},
			},
			TypeMeta: metav1.TypeMeta{
				Kind:       "PersistentVolume",
				APIVersion: "v1",
			},
			Spec: core.PersistentVolumeSpec{
				Capacity: resourceList,
				AccessModes: []core.PersistentVolumeAccessMode{
					core.ReadWriteOnce,
				},
				PersistentVolumeReclaimPolicy: core.PersistentVolumeReclaimDelete,
				PersistentVolumeSource: core.PersistentVolumeSource{
					HostPath: &core.HostPathVolumeSource{
						Path: volumeList.Volumes[volume].Mountpoint,
					},
				},
				ClaimRef:         persistentClaimReference,
				StorageClassName: "local",
			},
			Status: core.PersistentVolumeStatus{
				Phase: core.VolumeBound,
			},
		})
	}

	return core.PersistentVolumeList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "PersistentVolumeList",
			APIVersion: "v1",
		},
		Items: persistentVolumes,
	}, nil
}
