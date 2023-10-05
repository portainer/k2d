package configmaps

import (
	"github.com/emicklei/go-restful/v3"
	"github.com/portainer/k2d/internal/adapter"
	"github.com/portainer/k2d/internal/api/utils"
	"github.com/portainer/k2d/internal/controller"
)

type ConfigMapService struct {
	adapter    *adapter.KubeDockerAdapter
	operations chan controller.Operation
}

func NewConfigMapService(adapter *adapter.KubeDockerAdapter, operations chan controller.Operation) ConfigMapService {
	return ConfigMapService{
		adapter:    adapter,
		operations: operations,
	}
}

func (svc ConfigMapService) RegisterConfigMapAPI(ws *restful.WebService) {
	configMapGVKExtension := map[string]string{
		"group":   "",
		"kind":    "ConfigMap",
		"version": "v1",
	}

	ws.Route(ws.POST("/v1/configmaps").
		To(svc.CreateConfigMap).
		Param(ws.QueryParameter("dryRun", "when present, indicates that modifications should not be persisted").DataType("string")))

	ws.Route(ws.POST("/v1/namespaces/{namespace}/configmaps").
		Filter(utils.NamespaceValidation(svc.adapter)).
		To(svc.CreateConfigMap).
		Param(ws.PathParameter("namespace", "namespace name").DataType("string")).
		Param(ws.QueryParameter("dryRun", "when present, indicates that modifications should not be persisted").DataType("string")))

	ws.Route(ws.GET("/v1/configmaps").
		To(svc.ListConfigMaps))

	ws.Route(ws.GET("/v1/namespaces/{namespace}/configmaps").
		Filter(utils.NamespaceValidation(svc.adapter)).
		Param(ws.PathParameter("namespace", "namespace name").DataType("string")).
		To(svc.ListConfigMaps))

	ws.Route(ws.DELETE("/v1/configmaps/{name}").
		To(svc.DeleteConfigMap).
		Param(ws.PathParameter("name", "name of the configmap").DataType("string")))

	ws.Route(ws.DELETE("/v1/namespaces/{namespace}/configmaps/{name}").
		Filter(utils.NamespaceValidation(svc.adapter)).
		To(svc.DeleteConfigMap).
		Param(ws.PathParameter("namespace", "namespace name").DataType("string")).
		Param(ws.PathParameter("name", "name of the configmap").DataType("string")))

	ws.Route(ws.GET("/v1/configmaps/{name}").
		To(svc.GetConfigMap).
		Param(ws.PathParameter("name", "name of the configmap").DataType("string")))

	ws.Route(ws.GET("/v1/namespaces/{namespace}/configmaps/{name}").
		Filter(utils.NamespaceValidation(svc.adapter)).
		To(svc.GetConfigMap).
		Param(ws.PathParameter("namespace", "namespace name").DataType("string")).
		Param(ws.PathParameter("name", "name of the configmap").DataType("string")))

	ws.Route(ws.PATCH("/v1/configmaps/{name}").
		To(svc.PatchConfigMap).
		Param(ws.PathParameter("name", "name of the configmap").DataType("string")).
		Param(ws.QueryParameter("dryRun", "when present, indicates that modifications should not be persisted").DataType("string")).
		AddExtension("x-kubernetes-group-version-kind", configMapGVKExtension))

	ws.Route(ws.PATCH("/v1/namespaces/{namespace}/configmaps/{name}").
		Filter(utils.NamespaceValidation(svc.adapter)).
		To(svc.PatchConfigMap).
		Param(ws.PathParameter("namespace", "namespace name").DataType("string")).
		Param(ws.PathParameter("name", "name of the configmap").DataType("string")).
		Param(ws.QueryParameter("dryRun", "when present, indicates that modifications should not be persisted").DataType("string")).
		AddExtension("x-kubernetes-group-version-kind", configMapGVKExtension))
}
