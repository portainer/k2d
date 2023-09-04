package adapter

import (
	"context"
	"fmt"
	"time"

	"github.com/docker/docker/client"
	"github.com/portainer/k2d/internal/adapter/converter"
	"github.com/portainer/k2d/internal/adapter/store"
	"github.com/portainer/k2d/internal/adapter/store/filesystem"
	"github.com/portainer/k2d/internal/adapter/store/volume"
	k2dtypes "github.com/portainer/k2d/internal/adapter/types"
	"github.com/portainer/k2d/internal/config"
	"github.com/portainer/k2d/internal/types"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/kubernetes/pkg/apis/apps"
	appsv1 "k8s.io/kubernetes/pkg/apis/apps/v1"
	"k8s.io/kubernetes/pkg/apis/core"
	corev1 "k8s.io/kubernetes/pkg/apis/core/v1"
)

type (
	// KubeDockerAdapter serves as a bridge between the Docker API and Kubernetes resources.
	// This struct performs multiple roles:
	// - Interacts with the Docker API: It uses the Docker client to perform operations like
	//   pulling images, starting containers, and more.
	//
	// - Converts Kubernetes Objects: It utilizes a conversion scheme to translate Kubernetes
	//   objects into their corresponding Docker objects, supporting multiple Kubernetes versions.
	//
	// - ConfigMap and Secret storage: It manages the storage of ConfigMaps and Secrets. It
	//   supports multiple storage backends, including in-memory, Docker volumes and on-disk.
	//   This includes Kubernetes ConfigMaps, Secrets, and Registry Secrets.
	//
	// - Logging: For debugging and operational insight, it utilizes a logging framework.
	//
	// - Time-Tracking: The `startTime` field records when this adapter was initialized. This
	//   timestamp is used as the creation time for certain Kubernetes resources.
	//
	// - Server Configuration: Contains configuration related to the k2d server, which is used when
	//   creating certain resources.
	//
	// This struct is a comprehensive utility for managing the interactions between Docker and Kubernetes.
	KubeDockerAdapter struct {
		cli                    *client.Client
		configMapStore         store.ConfigMapStore
		converter              *converter.DockerAPIConverter
		conversionScheme       *runtime.Scheme
		k2dServerConfiguration *types.K2DServerConfiguration
		logger                 *zap.SugaredLogger
		registrySecretStore    store.SecretStore
		startTime              time.Time
		secretStore            store.SecretStore
	}

	// KubeDockerAdapterOptions represents options that can be used to configure a new KubeDockerAdapter
	KubeDockerAdapterOptions struct {
		// K2DConfig is the global configuration of k2d
		K2DConfig *config.Config
		// Logger is the logger that will be used by the adapter
		Logger *zap.SugaredLogger
		// K2DServerConfiguration is the configuration of the k2d HTTP server
		ServerConfiguration *types.K2DServerConfiguration
	}
)

// NewKubeDockerAdapter creates a new KubeDockerAdapter
func NewKubeDockerAdapter(options *KubeDockerAdapterOptions) (*KubeDockerAdapter, error) {
	cli, err := client.NewClientWithOpts(
		client.FromEnv,
		client.WithAPIVersionNegotiation(),
		client.WithTimeout(options.K2DConfig.DockerClientTimeout),
	)
	if err != nil {
		return nil, fmt.Errorf("unable to create docker client: %w", err)
	}

	storeOptions := store.StoreOptions{
		Backend:         options.K2DConfig.StoreBackend,
		RegistryBackend: options.K2DConfig.StoreRegistryBackend,
		Logger:          options.Logger,
		Filesystem: filesystem.FileSystemStoreOptions{
			DataPath: options.K2DConfig.DataPath,
		},
		Volume: volume.VolumeStoreOptions{
			DockerCli:     cli,
			CopyImageName: options.K2DConfig.StoreVolumeCopyImageName,
		},
	}

	configMapStore, secretStore, err := store.ConfigureStore(storeOptions)
	if err != nil {
		return nil, fmt.Errorf("unable to initialize store backends: %w", err)
	}

	registrySecretStore, err := store.ConfigureRegistrySecretStore(storeOptions, options.K2DConfig.DataPath)
	if err != nil {
		return nil, fmt.Errorf("unable to initialize registry secret store: %w", err)
	}

	scheme := runtime.NewScheme()

	apps.AddToScheme(scheme)
	appsv1.AddToScheme(scheme)
	core.AddToScheme(scheme)
	corev1.AddToScheme(scheme)

	return &KubeDockerAdapter{
		cli:                    cli,
		converter:              converter.NewDockerAPIConverter(configMapStore, secretStore, options.ServerConfiguration),
		conversionScheme:       scheme,
		configMapStore:         configMapStore,
		k2dServerConfiguration: options.ServerConfiguration,
		logger:                 options.Logger,
		registrySecretStore:    registrySecretStore,
		secretStore:            secretStore,
		startTime:              time.Now(),
	}, nil
}

// ConvertK8SResource is used to convert Kubernetes objects from versioned to internal and vice-versa.
// The conversion is necessary because different versions of the Kubernetes API have
// different representations for the same object, and some operations may require
// a specific version of an object.
//
// The conversion is performed using the conversionScheme of the KubeDockerAdapter,
// using the source object (src) as model and the result is stored in the destination object (dest).
//
// Parameters:
// src: The source object to be converted
// dest: The target object, into which the converted object will be stored
func (adapter *KubeDockerAdapter) ConvertK8SResource(src, dest interface{}) error {
	return adapter.conversionScheme.Convert(src, dest, nil)
}

// ProvisionSystemResources sets up the essential system resources required for the KubeDockerAdapter to operate.
// This function takes care of provisioning two namespaces ("default" and a custom k2d namespace),
// as well as storing service account secrets necessary for client authentication.
//
// Parameters:
// - ctx: Context for managing cancellations and timeouts.
// - tokenPath: File path where the service account token is stored.
// - sslCACertPath: File path where the SSL CA certificate is stored.
//
// The function performs the following steps in order:
// 1. Calls provisionNamespace() to create or verify the "default" namespace.
// 2. Calls provisionNamespace() to create or verify a custom k2d namespace.
// 3. Calls storeServiceAccountSecret() to store the service account token and SSL CA certificate at the provided paths.
//
// Error Handling:
// - If provisioning of either namespace fails, an error is returned detailing which namespace failed.
// - If storing the service account secret fails, an error is returned.
// - In each case, the underlying error is wrapped with additional context.
//
// Returns:
// - Returns nil if all resources are successfully provisioned.
// - Returns an error if any step in the provisioning process fails.
func (adapter *KubeDockerAdapter) ProvisionSystemResources(ctx context.Context, tokenPath, sslCACertPath string) error {
	err := adapter.provisionNamespace(ctx, "default")
	if err != nil {
		return fmt.Errorf("unable to provision default namespace: %w", err)
	}

	err = adapter.provisionNamespace(ctx, k2dtypes.K2DNamespaceName)
	if err != nil {
		return fmt.Errorf("unable to provision k2d namespace: %w", err)
	}

	err = adapter.storeServiceAccountSecret(tokenPath, sslCACertPath)
	if err != nil {
		return fmt.Errorf("unable to store service account secret: %w", err)
	}

	return nil
}
