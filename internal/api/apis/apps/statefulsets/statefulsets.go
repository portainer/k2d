package statefulsets

import (
	"github.com/emicklei/go-restful/v3"
	"github.com/portainer/k2d/internal/adapter"
	"github.com/portainer/k2d/internal/api/utils"
	"github.com/portainer/k2d/internal/controller"
)

type StatefulSetService struct {
	adapter    *adapter.KubeDockerAdapter
	operations chan controller.Operation
}

func NewStatefulSetService(adapter *adapter.KubeDockerAdapter, operations chan controller.Operation) StatefulSetService {
	return StatefulSetService{
		adapter:    adapter,
		operations: operations,
	}
}

func (svc StatefulSetService) RegisterStatefulSetAPI(ws *restful.WebService) {
	// This is required for kubectl to be able to use the --dry-run=server flag
	statefulSetGVKExtension := map[string]string{
		"group":   "apps",
		"kind":    "StatefulSet",
		"version": "v1",
	}

	ws.Route(ws.POST("/v1/statefulsets").
		To(svc.CreateStatefulSet).
		Param(ws.QueryParameter("dryRun", "when present, indicates that modifications should not be persisted").DataType("string")))

	ws.Route(ws.POST("/v1/namespaces/{namespace}/statefulsets").
		To(svc.CreateStatefulSet).
		Param(ws.QueryParameter("dryRun", "when present, indicates that modifications should not be persisted").DataType("string")))

	ws.Route(ws.GET("/v1/statefulsets").
		To(svc.ListStatefulSets))

	ws.Route(ws.GET("/v1/namespaces/{namespace}/statefulsets").
		To(svc.ListStatefulSets))

	ws.Route(ws.DELETE("/v1/statefulsets/{name}").
		To(svc.DeleteStatefulSet).
		Param(ws.PathParameter("name", "name of the statefulset").DataType("string")))

	ws.Route(ws.DELETE("/v1/namespaces/{namespace}/statefulsets/{name}").
		To(svc.DeleteStatefulSet).
		Param(ws.PathParameter("name", "name of the statefulset").DataType("string")))

	ws.Route(ws.GET("/v1/statefulsets/{name}").
		To(utils.UnsupportedOperation).
		Param(ws.PathParameter("name", "name of the statefulset").DataType("string")))

	ws.Route(ws.GET("/v1/namespaces/{namespace}/statefulsets/{name}").
		To(utils.UnsupportedOperation).
		Param(ws.PathParameter("name", "name of the statefulset").DataType("string")))

	ws.Route(ws.PATCH("/v1/statefulsets/{name}").
		To(utils.UnsupportedOperation).
		Param(ws.QueryParameter("dryRun", "when present, indicates that modifications should not be persisted").DataType("string")).
		AddExtension("x-kubernetes-group-version-kind", statefulSetGVKExtension).
		Param(ws.PathParameter("name", "name of the statefulset").DataType("string")))

	ws.Route(ws.PATCH("/v1/namespaces/{namespace}/statefulsets/{name}").
		To(utils.UnsupportedOperation).
		Param(ws.QueryParameter("dryRun", "when present, indicates that modifications should not be persisted").DataType("string")).
		AddExtension("x-kubernetes-group-version-kind", statefulSetGVKExtension).
		Param(ws.PathParameter("name", "name of the statefulset").DataType("string")))
}
