package types

const (
	// K2DNamespaceName is the name of the namespace where k2d resources are stored
	K2DNamespaceName = "k2d"

	// K2dServiceAccountSecretName is the name of the secret used to store the system service account token and CA
	// certificate. This secret contains everything needed to authenticate with the Kubernetes API server.
	K2dServiceAccountSecretName = "k2d-serviceaccount"
)
