package types

// K2dServiceAccountSecretName is the name of the secret used to store the system service account token and CA
// certificate. This secret contains everything needed to authenticate with the Kubernetes API server.
const K2dServiceAccountSecretName = "k2d-serviceaccount"
