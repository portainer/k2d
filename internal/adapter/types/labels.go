package types

// TODO: instead of using a constant to store the last applied config for each kind of resource
// see if we can introduce a generic constant such as LastAppliedConfigLabelKey

const (
	// NamespaceLastAppliedConfigLabelKey is the key used to store the namespace specific last applied configuration in the container labels
	NamespaceLastAppliedConfigLabelKey = "namespace.k2d.io/last-applied-configuration"

	// NamespaceNameLabelKey is the key used to store the namespace name associated to a Docker resource in its labels
	NamespaceNameLabelKey = "namespace.k2d.io/name"

	// NetworkNameLabelKey is the key used to store the network name in the container labels
	NetworkNameLabelKey = "networking.k2d.io/network-name"

	// PersistentVolumeClaimLastAppliedConfigLabelKey is the key used to store the service specific last applied configuration in the container labels
	PersistentVolumeClaimLastAppliedConfigLabelKey = "persistentvolumeclaim.k2d.io/last-applied-configuration"

	// PersistentVolumeClaimNameLabelKey is the key used to store the persistent volume name in the container labels
	PersistentVolumeClaimNameLabelKey = "storage.k2d.io/pvc-name"

	// PersistentVolumeNameLabelKey is the key used to store the persistent volume name in the container labels
	PersistentVolumeNameLabelKey = "storage.k2d.io/pv-name"

	// PodLastAppliedConfigLabelKey is the key used to store the pod specific last applied configuration in the container labels
	PodLastAppliedConfigLabelKey = "pod.k2d.io/last-applied-configuration"

	// ServiceLastAppliedConfigLabelKey is the key used to store the service specific last applied configuration in the container labels
	ServiceLastAppliedConfigLabelKey = "service.k2d.io/last-applied-configuration"

	// ServiceNameLabelKey is the key used to store the service name in the container labels
	ServiceNameLabelKey = "workload.k2d.io/service-name"

	// WorkloadLabelKey is the key used to store the workload type in the container labels
	WorkloadLabelKey = "workload.k2d.io/type"

	// WorkloadLastAppliedConfigLabelKey is the key used to store the last applied configuration in the container labels
	WorkloadLastAppliedConfigLabelKey = "workload.k2d.io/last-applied-configuration"

	// WorkloadNameLabelKey is the key used to store the workload name in the container labels
	WorkloadNameLabelKey = "workload.k2d.io/name"
)

const (
	// DeploymentWorkloadType is the label value used to identify a Deployment workload
	// It is stored on a container as a label and used to filter containers when listing deployments
	DeploymentWorkloadType = "deployment"
)
