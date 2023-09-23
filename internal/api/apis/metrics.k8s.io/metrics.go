package metrics

import (
	"github.com/emicklei/go-restful/v3"
	"github.com/portainer/k2d/internal/adapter"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type MetricsService struct {
	adapter *adapter.KubeDockerAdapter
}

func NewMetricsService(adapter *adapter.KubeDockerAdapter) MetricsService {
	return MetricsService{
		adapter: adapter,
	}
}

func (svc MetricsService) GetAPIVersions(r *restful.Request, w *restful.Response) {
	apiVersion := metav1.APIVersions{
		TypeMeta: metav1.TypeMeta{
			Kind: "APIVersions",
		},
		Versions: []string{"metrics.k8s.io/v1beta1"},
	}

	w.WriteAsJson(apiVersion)
}

func (svc MetricsService) ListAPIResources(r *restful.Request, w *restful.Response) {
	resourceList := metav1.APIResourceList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "APIResourceList",
			APIVersion: "v1beta1",
		},
		GroupVersion: "metrics.k8s.io",
		APIResources: []metav1.APIResource{
			{
				Kind:         "PodMetrics",
				SingularName: "",
				Name:         "pods",
				Verbs:        []string{"get", "list"},
				Namespaced:   true,
			},
		},
	}

	w.WriteAsJson(resourceList)
}

func (svc MetricsService) RegisterMetricsAPI(routes *restful.WebService) {
	routes.Route(routes.GET("/v1beta1/namespaces/{namespace}/pods/{name}").
		Param(routes.PathParameter("namespace", "namespace name").DataType("string")).
		Param(routes.PathParameter("name", "name of the pod").DataType("string")).
		To(svc.GetMetrics))
}
