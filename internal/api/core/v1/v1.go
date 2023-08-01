package v1

import (
	"github.com/emicklei/go-restful/v3"
	"github.com/portainer/k2d/internal/adapter"
	"github.com/portainer/k2d/internal/api/core/v1/configmaps"
	"github.com/portainer/k2d/internal/api/core/v1/events"
	"github.com/portainer/k2d/internal/api/core/v1/namespaces"
	"github.com/portainer/k2d/internal/api/core/v1/nodes"
	"github.com/portainer/k2d/internal/api/core/v1/pods"
	"github.com/portainer/k2d/internal/api/core/v1/secrets"
	"github.com/portainer/k2d/internal/api/core/v1/services"
	"github.com/portainer/k2d/internal/controller"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type V1Service struct {
	configMaps configmaps.ConfigMapService
	events     events.EventService
	namespaces namespaces.NamespaceService
	nodes      nodes.NodeService
	pods       pods.PodService
	secrets    secrets.SecretService
	services   services.ServiceService
}

func NewV1Service(adapter *adapter.KubeDockerAdapter, operations chan controller.Operation) V1Service {
	return V1Service{
		configMaps: configmaps.NewConfigMapService(adapter, operations),
		events:     events.NewEventService(adapter),
		namespaces: namespaces.NewNamespaceService(adapter),
		nodes:      nodes.NewNodeService(adapter),
		pods:       pods.NewPodService(adapter, operations),
		secrets:    secrets.NewSecretService(adapter, operations),
		services:   services.NewServiceService(adapter, operations),
	}
}

func (svc V1Service) GetAPIVersions(r *restful.Request, w *restful.Response) {
	apiVersion := metav1.APIVersions{
		TypeMeta: metav1.TypeMeta{
			Kind: "APIVersions",
		},
		Versions: []string{"v1"},
	}

	w.WriteAsJson(apiVersion)
}

func (svc V1Service) ListAPIResources(r *restful.Request, w *restful.Response) {
	resourceList := metav1.APIResourceList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "APIResourceList",
			APIVersion: "v1",
		},
		GroupVersion: "v1",
		APIResources: []metav1.APIResource{
			{
				Kind:         "Namespace",
				SingularName: "",
				Name:         "namespaces",
				Verbs:        []string{"list"},
				Namespaced:   false,
				ShortNames:   []string{"ns"},
			},
			{
				Kind:         "Pod",
				SingularName: "",
				Name:         "pods",
				Verbs:        []string{"create", "list", "delete", "get", "patch"},
				Namespaced:   true,
			},
			{
				Kind:         "Node",
				SingularName: "",
				Name:         "nodes",
				Verbs:        []string{"list"},
				Namespaced:   false,
			},
			{
				Kind:         "Service",
				SingularName: "",
				Name:         "services",
				Verbs:        []string{"create", "list", "delete", "get", "patch"},
				Namespaced:   true,
				ShortNames:   []string{"svc"},
			},
			{
				Kind:         "ConfigMap",
				SingularName: "",
				Name:         "configmaps",
				Verbs:        []string{"create", "list", "delete", "get", "patch"},
				Namespaced:   true,
				ShortNames:   []string{"cm"},
			},
			{
				Kind:         "Secret",
				SingularName: "",
				Name:         "secrets",
				Verbs:        []string{"create", "list", "delete", "get", "patch"},
				Namespaced:   true,
			},
			{
				Kind:         "Event",
				SingularName: "",
				Name:         "events",
				Verbs:        []string{"list"},
				Namespaced:   false,
			},
		},
	}

	w.WriteAsJson(resourceList)
}

func (svc V1Service) RegisterV1API(routes *restful.WebService) {
	// configmaps
	svc.configMaps.RegisterConfigMapAPI(routes)

	// events
	// note that this is the deprecated API endpoint but it is still used by some clients (Lens)
	// the new endpoint is /apis/events.k8s.io/v1/events
	svc.events.RegisterEventAPI(routes)

	// namespaces
	svc.namespaces.RegisterNamespaceAPI(routes)

	// nodes
	svc.nodes.RegisterNodeAPI(routes)

	// pods
	svc.pods.RegisterPodAPI(routes)

	// secrets
	svc.secrets.RegisterSecretAPI(routes)

	// services
	svc.services.RegisterServiceAPI(routes)
}
