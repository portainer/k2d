package types

const (
	// WorkloadLabelKey is the key used to store the workload type in the container labels
	WorkloadLabelKey = "workload.k2d.io/type"
	// WorkloadLabelValueDeployment is the key used to store the last applied configuration in the container labels
	WorkloadLastAppliedConfigLabelKey = "workload.k2d.io/last-applied-configuration"
)
