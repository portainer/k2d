package events

import (
	"github.com/emicklei/go-restful/v3"
	"github.com/portainer/k2d/internal/adapter"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type EventsService struct {
	adapter *adapter.KubeDockerAdapter
}

func NewEventsService(adapter *adapter.KubeDockerAdapter) EventsService {
	return EventsService{
		adapter: adapter,
	}
}

func (svc EventsService) GetAPIVersions(r *restful.Request, w *restful.Response) {
	apiVersion := metav1.APIVersions{
		TypeMeta: metav1.TypeMeta{
			Kind: "APIVersions",
		},
		Versions: []string{"events.k8s.io/v1"},
	}

	w.WriteAsJson(apiVersion)
}

func (svc EventsService) ListAPIResources(r *restful.Request, w *restful.Response) {
	resourceList := metav1.APIResourceList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "APIResourceList",
			APIVersion: "v1",
		},
		GroupVersion: "events.k8s.io/v1",
		APIResources: []metav1.APIResource{
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

func (svc EventsService) RegisterEventAPI(routes *restful.WebService) {
	// events
	routes.Route(routes.GET("/v1/events").
		To(svc.ListEvents))

	routes.Route(routes.GET("/v1/namespaces/{namespace}/events").
		To(svc.ListEvents))
}
