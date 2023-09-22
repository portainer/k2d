package types

const (
	// LastAppliedConfigLabelKey is the key used to store the last applied configuration of a resource in the container labels
	// It can be used to retrieve the original workload definition (deployment, pod) from a container or the resource definition of resource kinds (namespace, persistent volume claim)
	LastAppliedConfigLabelKey = "resource.k2d.io/last-applied-configuration"

	// NamespaceNameLabelKey is the key used to store the namespace name associated to a Docker resource in its labels
	NamespaceNameLabelKey = "resource.k2d.io/namespace-name"

	// PodLastAppliedConfigLabelKey is the key used to store the pod definition in the container labels
	// It can be used to retrieve the pod definition from a container created via a deployment
	PodLastAppliedConfigLabelKey = "resource.k2d.io/pod/last-applied-configuration"

	// ServiceLastAppliedConfigLabelKey is the key used to store the service definition associated to a workload in the container labels
	ServiceLastAppliedConfigLabelKey = "resource.k2d.io/service/last-applied-configuration"
)

const (
	// NetworkNameLabelKey is the key used to store the network name in the container labels
	NetworkNameLabelKey = "networking.k2d.io/network-name"
)

const (
	// PersistentVolumeClaimNameLabelKey is the key used to store the persistent volume claim name in the labels of a system configmap
	PersistentVolumeClaimNameLabelKey = "storage.k2d.io/pvc-name"

	// PersistentVolumeNameLabelKey is the key used to store the persistent volume name in the labels of a system configmap or a Docker volume
	PersistentVolumeNameLabelKey = "storage.k2d.io/pv-name"

	// PersistentVolumeClaimTargetNamespaceLabelKey is the key used to store the target namespace of a persistent volume claim in the labels of a system configmap
	// This is used to identify the namespace where the persistent volume claim is used (e.g. the namespace of the workload)
	PersistentVolumeClaimTargetNamespaceLabelKey = "storage.k2d.io/pvc-target-namespace"

	// StorageTypeLabelKey is the key used to store the storage type in the labels of a system configmap or a Docker volume
	// It is used to differentiate between persistent volumes and config maps when listing volumes
	StorageTypeLabelKey = "storage.k2d.io/type"
)

const (
	// PersistentVolumeStorageType is the label value used to identify a persistent volume storage
	// It is stored inside metadata as a label and used to filter persistent volumes when listing persistent volumes
	PersistentVolumeStorageType = "pv"
)

const (
	// ServiceNameLabelKey is the key used to store the service name associated to the workload in the container labels
	ServiceNameLabelKey = "workload.k2d.io/service-name"

	// WorkloadTypeLabelKey is the key used to store the workload type in the container labels
	WorkloadTypeLabelKey = "workload.k2d.io/type"

	// WorkloadNameLabelKey is the key used to store the workload name in the container labels
	WorkloadNameLabelKey = "workload.k2d.io/name"
)

const (
	// DeploymentWorkloadType is the label value used to identify a Deployment workload
	// It is stored on a container as a label and used to filter containers when listing deployments
	DeploymentWorkloadType = "deployment"
)
