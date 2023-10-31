package types

// K2DServerConfiguration represents the configuration of the k2d server
type K2DServerConfiguration struct {
	// ServerIpAddr is the IP address on which the k2d server listens. It will be shared with all created containers through
	// the KUBERNETES_SERVICE_HOST environment variable
	ServerIpAddr string
	// ServerPort is the port on which the k2d server listens. It will be shared with all created containers through
	// the KUBERNETES_SERVICE_PORT environment variable
	ServerPort int
	// CaPath is the path to the CA certificate that is used to sign the server certificate. It will be mounted into all
	// containers
	CaPath string
	// TokenPath is the path to the token file that will be mounted into all containers
	TokenPath string
	// Secret is the secret used to protect some API operations such as getting the kubeconfig.
	Secret string
}

const (
	// Version represents the k2d server version
	Version = "1.0.0"
	// RequestIDHeader is the name of the header that contains the request ID used for tracing purposes
	RequestIDHeader = "X-K2d-Request-Id"
)
