package converter

import (
	"fmt"
	"time"

	"github.com/docker/docker/api/types"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/kubernetes/pkg/apis/core"
)

func (converter *DockerAPIConverter) ConvertInfoVersionToNodes(info types.Info, version types.Version, startTime time.Time) core.NodeList {
	return core.NodeList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "NodeList",
			APIVersion: "v1",
		},
		Items: []core.Node{
			{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Node",
					APIVersion: "v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: info.Name,
					UID:  k8stypes.UID(info.ID),
					CreationTimestamp: metav1.Time{
						Time: startTime,
					},
					Labels: map[string]string{
						"beta.kubernetes.io/arch":        info.Architecture,
						"beta.kubernetes.io/os":          info.OSType,
						"kubernetes.io/arch":             info.Architecture,
						"kubernetes.io/hostname":         info.Name,
						"kubernetes.io/os":               info.OSType,
						"node-role.kubernetes.io/master": "",
					},
				},
				Spec: core.NodeSpec{
					ProviderID: "k2d",
				},
				Status: core.NodeStatus{
					Conditions: []core.NodeCondition{
						{
							Type:               "Ready",
							Status:             "True",
							Reason:             "KubeletReady",
							Message:            "kubelet is posting ready status",
							LastHeartbeatTime:  metav1.NewTime(time.Now()),
							LastTransitionTime: metav1.NewTime(time.Now()),
						},
					},
					NodeInfo: core.NodeSystemInfo{
						Architecture:            info.Architecture,
						ContainerRuntimeVersion: version.Version,
						KernelVersion:           info.KernelVersion,
						KubeletVersion:          fmt.Sprintf("docker-%s", version.Version),
						MachineID:               info.ID,
						OperatingSystem:         info.OSType,
						SystemUUID:              info.ID,
					},
					Capacity: core.ResourceList{
						core.ResourceCPU:    *resource.NewQuantity(int64(info.NCPU), resource.DecimalSI),
						core.ResourceMemory: *resource.NewQuantity(int64(info.MemTotal), resource.BinarySI),
					},
				},
			},
		},
	}
}
