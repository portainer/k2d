package daemonsets

import (
	"github.com/emicklei/go-restful/v3"
	"github.com/portainer/k2d/internal/adapter"
	"github.com/portainer/k2d/internal/api/utils"
	"github.com/portainer/k2d/internal/controller"
)

type DaemonSetService struct {
	adapter    *adapter.KubeDockerAdapter
	operations chan controller.Operation
}

func NewDaemonSetService(adapter *adapter.KubeDockerAdapter, operations chan controller.Operation) DaemonSetService {
	return DaemonSetService{
		adapter:    adapter,
		operations: operations,
	}
}

func (svc DaemonSetService) RegisterDaemonSetAPI(ws *restful.WebService) {
	// This is required for kubectl to be able to use the --dry-run=server flag
	daemonSetGVKExtension := map[string]string{
		"group":   "apps",
		"kind":    "DaemonSet",
		"version": "v1",
	}

	ws.Route(ws.POST("/v1/daemonsets").
		To(svc.CreateDaemonSet).
		Param(ws.QueryParameter("dryRun", "when present, indicates that modifications should not be persisted").DataType("string")))

	ws.Route(ws.POST("/v1/namespaces/{namespace}/daemonsets").
		To(svc.CreateDaemonSet).
		Param(ws.QueryParameter("dryRun", "when present, indicates that modifications should not be persisted").DataType("string")))

	ws.Route(ws.GET("/v1/daemonsets").
		To(svc.ListDaemonSets))

	ws.Route(ws.GET("/v1/namespaces/{namespace}/daemonsets").
		To(svc.ListDaemonSets))

	ws.Route(ws.DELETE("/v1/daemonsets/{name}").
		To(svc.DeleteDaemonSet).
		Param(ws.PathParameter("name", "name of the daemonset").DataType("string")))

	ws.Route(ws.DELETE("/v1/namespaces/{namespace}/daemonsets/{name}").
		To(svc.DeleteDaemonSet).
		Param(ws.PathParameter("name", "name of the daemonset").DataType("string")))

	ws.Route(ws.GET("/v1/daemonsets/{name}").
		To(utils.UnsupportedOperation).
		Param(ws.PathParameter("name", "name of the daemonset").DataType("string")))

	ws.Route(ws.GET("/v1/namespaces/{namespace}/daemonsets/{name}").
		To(utils.UnsupportedOperation).
		Param(ws.PathParameter("name", "name of the daemonset").DataType("string")))

	ws.Route(ws.PATCH("/v1/daemonsets/{name}").
		To(utils.UnsupportedOperation).
		Param(ws.QueryParameter("dryRun", "when present, indicates that modifications should not be persisted").DataType("string")).
		AddExtension("x-kubernetes-group-version-kind", daemonSetGVKExtension).
		Param(ws.PathParameter("name", "name of the daemonset").DataType("string")))

	ws.Route(ws.PATCH("/v1/namespaces/{namespace}/daemonsets/{name}").
		To(utils.UnsupportedOperation).
		Param(ws.QueryParameter("dryRun", "when present, indicates that modifications should not be persisted").DataType("string")).
		AddExtension("x-kubernetes-group-version-kind", daemonSetGVKExtension).
		Param(ws.PathParameter("name", "name of the daemonset").DataType("string")))
}
