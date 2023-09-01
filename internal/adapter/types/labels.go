package types

const (
	// NetworkNameLabelKey is the key used to store the network name in the container labels
	NetworkNameLabelKey = "networking.k2d.io/network-name"

	// NamespaceLabelKey is the key used to store the namespace name associated to a Docker resource in its labels
	NamespaceLabelKey = "namespace.k2d.io/name"

	// NamespaceLastAppliedConfigLabelKey is the key used to store the namespace specific last applied configuration in the container labels
	NamespaceLastAppliedConfigLabelKey = "namespace.k2d.io/last-applied-configuration"

	// PodLastAppliedConfigLabelKey is the key used to store the pod specific last applied configuration in the container labels
	PodLastAppliedConfigLabelKey = "pod.k2d.io/last-applied-configuration"

	// ServiceLastAppliedConfigLabelKey is the key used to store the service specific last applied configuration in the container labels
	ServiceLastAppliedConfigLabelKey = "service.k2d.io/last-applied-configuration"

	// PersistentVolumeClaimLastAppliedConfigLabelKey is the key used to store the service specific last applied configuration in the container labels
	PersistentVolumeClaimLastAppliedConfigLabelKey = "persistentvolumeclaim.k2d.io/last-applied-configuration"

	// ServiceNameLabelKey is the key used to store the service name in the container labels
	ServiceNameLabelKey = "workload.k2d.io/service-name"

	// WorkloadLabelKey is the key used to store the workload type in the container labels
	WorkloadLabelKey = "workload.k2d.io/type"

	// WorkloadLastAppliedConfigLabelKey is the key used to store the last applied configuration in the container labels
	WorkloadLastAppliedConfigLabelKey = "workload.k2d.io/last-applied-configuration"

	// WorkloadNameLabelKey is the key used to store the workload name in the container labels
	WorkloadNameLabelKey = "workload.k2d.io/name"

	// PersistentVolumeLabelKey is the key used to store the persistent volume name in the container labels
	PersistentVolumeLabelKey = "storage.k2d.io/pv"

	// PersistentVolumeClaimLabelKey is the key used to store the persistent volume name in the container labels
	PersistentVolumeClaimLabelKey = "storage.k2d.io/pvc"
)
