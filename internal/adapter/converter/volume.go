package converter

// import (
// 	"github.com/docker/docker/api/types"
// 	k2dtypes "github.com/portainer/k2d/internal/adapter/types"
// 	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
// 	"k8s.io/kubernetes/pkg/apis/core"
// )

// func (converter *DockerAPIConverter) ConvertVolumeToPersistentVolume(namespaceName string, network types.NetworkResource) core.PersistentVolume {
// 	return core.PersistentVolume{
// 		ObjectMeta: metav1.ObjectMeta{
// 			Name:      buildPersistentVolumeName(network.Name, namespaceName),
// 			Namespace: namespaceName,
// 			Labels: map[string]string{
// 				k2dtypes.PersistentVolumeLabelKey: buildPersistentVolumeName(network.Name, namespaceName),
// 			},
// 		},
// 		Spec: core.PersistentVolumeSpec{
// 			AccessModes: []core.PersistentVolumeAccessMode{
// 				core.ReadWriteOnce,
// 			},
// 			Capacity: core.ResourceList{
// 				core.ResourceStorage: network.DriverOpts["size"],
// 			},
// 			PersistentVolumeSource: core.PersistentVolumeSource{
// 				HostPath: &core.HostPathVolumeSource{
// 					Path: network.Name,
// 				},
// 			},
// 			PersistentVolumeReclaimPolicy: core.PersistentVolumeReclaimDelete,
// 			StorageClassName:              "local",
// 		},
// 	}
// }
