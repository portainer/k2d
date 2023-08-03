package adapter

import (
	"fmt"
	"time"

	"github.com/docker/docker/client"
	"github.com/portainer/k2d/internal/adapter/converter"
	"github.com/portainer/k2d/internal/adapter/filesystem"
	"github.com/portainer/k2d/internal/types"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/kubernetes/pkg/apis/apps"
	appsv1 "k8s.io/kubernetes/pkg/apis/apps/v1"
	"k8s.io/kubernetes/pkg/apis/core"
	corev1 "k8s.io/kubernetes/pkg/apis/core/v1"
)

type (
	// KubeDockerAdapter is used to interact with the Docker API and to convert Kubernetes objects to Docker objects
	// It stores some Kubernetes objects in a filesystem store.
	// It contains a conversion scheme that is used to convert Kubernetes across different versions.
	// It registers its start time to be used as the creation timestamp of some Kubernetes objects such as the default namespace
	// and the (single) Kubernetes node.
	// It also contains the configuration of the k2d server to be used by some resources that are created by the adapter.
	KubeDockerAdapter struct {
		cli                    *client.Client
		converter              *converter.DockerAPIConverter
		fileSystemStore        *filesystem.FileSystemStore
		logger                 *zap.SugaredLogger
		conversionScheme       *runtime.Scheme
		startTime              time.Time
		k2dServerConfiguration *types.K2DServerConfiguration
	}

	// KubeDockerAdapterOptions represents options that can be used to configure a new KubeDockerAdapter
	KubeDockerAdapterOptions struct {
		// DataPath is the path to the data directory where the configmaps and secrets will be stored
		DataPath string
		// VolumePath is the path to the directory where the volumes will be stored
		VolumePath string
		// DockerClientTimeout is the timeout that will be used when communicating with the Docker API with the Docker client
		// It is responsible for the timeout of the Docker API calls such as creating a container, pulling an image...
		DockerClientTimeout time.Duration
		// K2DServerConfiguration is the configuration of the k2d server
		ServerConfiguration *types.K2DServerConfiguration
		// Logger is the logger that will be used by the adapter
		Logger *zap.SugaredLogger
	}
)

// NewKubeDockerAdapter creates a new KubeDockerAdapter
func NewKubeDockerAdapter(options *KubeDockerAdapterOptions) (*KubeDockerAdapter, error) {
	cli, err := client.NewClientWithOpts(
		client.FromEnv,
		client.WithAPIVersionNegotiation(),
		client.WithTimeout(options.DockerClientTimeout),
	)
	if err != nil {
		return nil, fmt.Errorf("unable to create docker client: %w", err)
	}

	filesystemStore, err := filesystem.NewFileSystemStore(options.VolumePath)
	if err != nil {
		return nil, fmt.Errorf("unable to create filesystem store: %w", err)
	}

	scheme := runtime.NewScheme()

	apps.AddToScheme(scheme)
	appsv1.AddToScheme(scheme)
	core.AddToScheme(scheme)
	corev1.AddToScheme(scheme)

	return &KubeDockerAdapter{
		cli:                    cli,
		converter:              converter.NewDockerAPIConverter(filesystemStore, options.ServerConfiguration),
		fileSystemStore:        filesystemStore,
		logger:                 options.Logger,
		conversionScheme:       scheme,
		startTime:              time.Now(),
		k2dServerConfiguration: options.ServerConfiguration,
	}, nil
}

// ConvertObjectToVersionedObject is used to convert Kubernetes objects from one version to another.
// The conversion is necessary because different versions of the Kubernetes API have
// different representations for the same object, and some operations may require
// a specific version of an object.
//
// This function is specifically designed to handle the conversion of generic
// runtime.Object types, which are used to represent Kubernetes objects in a version-agnostic manner.
// The source object (src) is expected to be of this type.
//
// It first asserts that the source object is indeed a runtime.Object.
// If this assertion fails, an error is returned.
//
// Then, it determines the GroupVersion of the source object by calling its GetObjectKind
// method and extracting the GroupVersionKind. A GroupVersioner is then constructed
// from this GroupVersion.
//
// Finally, the conversion is performed using the conversionScheme of the KubeDockerAdapter,
// and the result is stored in the destination object (dest).
//
// Parameters:
// src: The source object to be converted, expected to be a runtime.Object.
// dest: The target object, into which the converted object will be stored. Should also be a runtime.Object.
//
// Returns:
// An error if the conversion fails or if the source object is not a runtime.Object.
// If the conversion is successful, returns nil.
func (adapter *KubeDockerAdapter) ConvertObjectToVersionedObject(src, dest interface{}) error {
	k8sObject, ok := src.(runtime.Object)
	if !ok {
		return fmt.Errorf("unable to convert src to runtime.Object")
	}

	gvs := runtime.GroupVersioner(schema.GroupVersions([]schema.GroupVersion{k8sObject.GetObjectKind().GroupVersionKind().GroupVersion()}))

	return adapter.conversionScheme.Convert(src, dest, gvs)
}
