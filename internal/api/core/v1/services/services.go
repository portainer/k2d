package services

import (
	"github.com/emicklei/go-restful/v3"
	"github.com/portainer/k2d/internal/adapter"
	"github.com/portainer/k2d/internal/api/utils"
	"github.com/portainer/k2d/internal/controller"
)

type ServiceService struct {
	adapter    *adapter.KubeDockerAdapter
	operations chan controller.Operation
}

func NewServiceService(adapter *adapter.KubeDockerAdapter, operations chan controller.Operation) ServiceService {
	return ServiceService{
		adapter:    adapter,
		operations: operations,
	}
}

func (svc ServiceService) RegisterServiceAPI(ws *restful.WebService) {
	serviceGVKExtension := map[string]string{
		"group":   "",
		"kind":    "Service",
		"version": "v1",
	}

	ws.Route(ws.POST("/v1/services").
		To(svc.CreateService).
		Param(ws.QueryParameter("dryRun", "when present, indicates that modifications should not be persisted").DataType("string")))

	ws.Route(ws.POST("/v1/namespaces/{namespace}/services").
		Filter(utils.NamespaceValidation(svc.adapter)).
		To(svc.CreateService).
		Param(ws.PathParameter("namespace", "namespace name").DataType("string")).
		Param(ws.QueryParameter("dryRun", "when present, indicates that modifications should not be persisted").DataType("string")))

	ws.Route(ws.GET("/v1/services").
		To(svc.ListServices))

	ws.Route(ws.GET("/v1/namespaces/{namespace}/services").
		Filter(utils.NamespaceValidation(svc.adapter)).
		To(svc.ListServices).
		Param(ws.PathParameter("namespace", "namespace name").DataType("string")))

	ws.Route(ws.DELETE("/v1/services/{name}").
		To(svc.DeleteService).
		Param(ws.PathParameter("name", "name of the service").DataType("string")))

	ws.Route(ws.DELETE("/v1/namespaces/{namespace}/services/{name}").
		Filter(utils.NamespaceValidation(svc.adapter)).
		To(svc.DeleteService).
		Param(ws.PathParameter("namespace", "namespace name").DataType("string")).
		Param(ws.PathParameter("name", "name of the service").DataType("string")))

	ws.Route(ws.GET("/v1/services/{name}").
		To(svc.GetService).
		Param(ws.PathParameter("name", "name of the service").DataType("string")))

	ws.Route(ws.GET("/v1/namespaces/{namespace}/services/{name}").
		Filter(utils.NamespaceValidation(svc.adapter)).
		To(svc.GetService).
		Param(ws.PathParameter("namespace", "namespace name").DataType("string")).
		Param(ws.PathParameter("name", "name of the service").DataType("string")))

	ws.Route(ws.PATCH("/v1/services/{name}").
		To(svc.PatchService).
		Param(ws.PathParameter("name", "name of the service").DataType("string")).
		Param(ws.QueryParameter("dryRun", "when present, indicates that modifications should not be persisted").DataType("string")).
		AddExtension("x-kubernetes-group-version-kind", serviceGVKExtension))

	ws.Route(ws.PATCH("/v1/namespaces/{namespace}/services/{name}").
		Filter(utils.NamespaceValidation(svc.adapter)).
		To(svc.PatchService).
		Param(ws.PathParameter("namespace", "namespace name").DataType("string")).
		Param(ws.PathParameter("name", "name of the service").DataType("string")).
		Param(ws.QueryParameter("dryRun", "when present, indicates that modifications should not be persisted").DataType("string")).
		AddExtension("x-kubernetes-group-version-kind", serviceGVKExtension))
}
