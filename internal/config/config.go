package config

import "time"

// Config represents the configuration of the k2d application.
type Config struct {
	// AdvertiseAddr represents the advertised address for the application.
	// This address is used to generate a certificate for the k2d API server that Kubernetes clients
	// (such as kubectl) can use to connect to it.
	// It is expected to be provided through an environment variable named K2D_ADVERTISE_ADDR.
	AdvertiseAddr string `env:"K2D_ADVERTISE_ADDR"`

	// DataPath represents the path for application data storage.
	// If not provided through an environment variable named K2D_DATA_PATH,
	// the default value is set to /var/lib/k2d.
	DataPath string `env:"K2D_DATA_PATH,default=/var/lib/k2d"`

	// DockerClientTimeout represents the timeout duration for Docker client operations.
	// If not provided through an environment variable named K2D_DOCKER_CLIENT_TIMEOUT,
	// the default value is set to 10 minutes (10m).
	DockerClientTimeout time.Duration `env:"K2D_DOCKER_CLIENT_TIMEOUT,default=10m"`

	// LogFormat represents the log format for the application.
	// If not provided through an environment variable named K2D_LOG_FORMAT,
	// the default value is set to text.
	// Valid values are: text, json.
	LogFormat string `env:"K2D_LOG_FORMAT,default=text"`

	// LogLevel represents the log level for the application.
	// If not provided through an environment variable named K2D_LOG_LEVEL,
	// the default value is set to debug.
	LogLevel string `env:"K2D_LOG_LEVEL,default=debug"`

	// OperationBatchMaxSize represents the maximum number of operations to process in a single batch.
	// If not provided through an environment variable named K2D_OPERATION_BATCH_MAX_SIZE,
	// the default value is set to 25.
	OperationBatchMaxSize int `env:"K2D_OPERATION_BATCH_MAX_SIZE,default=25"`

	// Port represents the port number for the application.
	// If not provided through an environment variable named K2D_PORT,
	// the default value is set to 6443.
	Port int `env:"K2D_PORT,default=6443"`

	// PortainerAgentVersion represents the version of the Portainer agent to deploy.
	// If not provided through an environment variable named PORTAINER_AGENT_VERSION,
	// the default value is set to latest.
	PortainerAgentVersion string `env:"PORTAINER_AGENT_VERSION,default=latest"`

	// PortainerEdgeKey represents the key used to automatically deploy the Portainer Edge agent
	// (async) as part of the k2d initialization process.
	// It is optional and the agent will only be deployed if the PORTAINER_EDGE_KEY environment variable
	// is provided.
	PortainerEdgeKey string `env:"PORTAINER_EDGE_KEY"`

	// PortainerEdgeID represents the unique Edge ID associated with the Portainer Edge agent.
	// If it is not provided through an environment variable named PORTAINER_EDGE_ID,
	// a random ID will be generated.
	PortainerEdgeID string `env:"PORTAINER_EDGE_ID"`

	// Secret represents the secret used to protect some API operations such as getting
	// the kubeconfig. If it is not provided through an environment variable named K2D_SECRET,
	// a random secret will be generated.
	Secret string `env:"K2D_SECRET"`

	// StoreBackend represents the backend used to store secrets and configmaps.
	// If not provided through an environment variable named K2D_STORE_BACKEND,
	// the default value is set to disk.
	StoreBackend string `env:"K2D_STORE_BACKEND,default=disk"`

	// StoreVolumeCopyImageName represents the name of the container image used to copy and read from volumes
	// when using the volume store for secrets and configmaps.
	// If not provided through an environment variable named K2D_STORE_VOLUME_COPY_IMAGE_NAME,
	// the default value is set to docker.io/library/alpine:latest.
	StoreVolumeCopyImageName string `env:"K2D_STORE_VOLUME_COPY_IMAGE_NAME,default=docker.io/library/alpine:latest"`
}
