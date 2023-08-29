package types

const (
	// WorkloadLabelKey is the key used to store the workload type in the container labels
	WorkloadLabelKey = "workload.k2d.io/type"
	// WorkloadLabelValueDeployment is the key used to store the last applied configuration in the container labels
	WorkloadLastAppliedConfigLabelKey = "workload.k2d.io/last-applied-configuration"
	// ServiceLastAppliedConfigLabelKey is the key used to store the service specific last applied configuration in the container labels
	ServiceLastAppliedConfigLabelKey = "service.k2d.io/last-applied-configuration"
	// PodLastAppliedConfigLabelKey is the key used to store the pod specific last applied configuration in the container labels
	PodLastAppliedConfigLabelKey = "pod.k2d.io/last-applied-configuration"
	// ServiceNameLabelKey is the key used to store the service name in the container labels
	ServiceNameLabelKey = "workload.k2d.io/service-name"
	// ServiceStatusErrorMessage is the key used to store the service status error message in the container labels
	ServiceStatusErrorMessage = "service.k2d.io/status-error-message"
)
