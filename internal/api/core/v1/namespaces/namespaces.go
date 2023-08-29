package namespaces

import (
	"github.com/emicklei/go-restful/v3"
	"github.com/portainer/k2d/internal/adapter"
	"github.com/portainer/k2d/internal/controller"
)

type NamespaceService struct {
	adapter    *adapter.KubeDockerAdapter
	operations chan controller.Operation
}

func NewNamespaceService(adapter *adapter.KubeDockerAdapter, operations chan controller.Operation) NamespaceService {
	return NamespaceService{
		adapter:    adapter,
		operations: operations,
	}
}

func (svc NamespaceService) RegisterNamespaceAPI(ws *restful.WebService) {
	namespaceGVKExtension := map[string]string{
		"group":   "",
		"kind":    "Namespace",
		"version": "v1",
	}

	ws.Route(ws.POST("/v1/namespaces").
		To(svc.CreateNamespace).
		Param(ws.QueryParameter("dryRun", "when present, indicates that modifications should not be persisted").DataType("string")))

	ws.Route(ws.GET("/v1/namespaces").
		To(svc.ListNamespaces))

	// TODO: check if watch is required
	ws.Route(ws.GET("/v1/namespaces/{name}").
		To(svc.GetNamespace).
		Param(ws.PathParameter("name", "name of the namespace").DataType("string")).
		Param(ws.PathParameter("watch", "watch for changes").DataType("boolean")))

	ws.Route(ws.PATCH("/v1/namespaces/{name}").
		To(svc.PatchNamespace).
		Param(ws.PathParameter("name", "name of the namespace").DataType("string")).
		Param(ws.QueryParameter("dryRun", "when present, indicates that modifications should not be persisted").DataType("string")).
		AddExtension("x-kubernetes-group-version-kind", namespaceGVKExtension))

	ws.Route(ws.DELETE("/v1/namespaces/{name}").
		To(svc.DeleteNamespace).
		Param(ws.PathParameter("name", "name of the namespace").DataType("string")))
}
