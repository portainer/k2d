package types

// TODO: find a better way to organize this package

const (
	// NamespaceLastAppliedConfigLabelKey is the key used to store the namespace specific last applied configuration in the container labels
	NamespaceLastAppliedConfigLabelKey = "namespace.k2d.io/last-applied-configuration"
	// NamespaceLabelKey is the key used to store the namespace name associated to a Docker resource in its labels
	NamespaceLabelKey = "namespace.k2d.io/name"

	// K2DNamespaceName is the name of the namespace where k2d resources are stored
	K2DNamespaceName = "k2d"
)
