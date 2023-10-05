package pods

import (
	"github.com/emicklei/go-restful/v3"
	"github.com/portainer/k2d/internal/adapter"
	"github.com/portainer/k2d/internal/api/utils"
	"github.com/portainer/k2d/internal/controller"
)

type PodService struct {
	adapter    *adapter.KubeDockerAdapter
	operations chan controller.Operation
}

func NewPodService(adapter *adapter.KubeDockerAdapter, operations chan controller.Operation) PodService {
	return PodService{
		adapter:    adapter,
		operations: operations,
	}
}

func (svc PodService) RegisterPodAPI(ws *restful.WebService) {
	podGVKExtension := map[string]string{
		"group":   "",
		"kind":    "Pod",
		"version": "v1",
	}

	ws.Route(ws.POST("/v1/pods").
		To(svc.CreatePod).
		Param(ws.QueryParameter("dryRun", "when present, indicates that modifications should not be persisted").DataType("string")))

	ws.Route(ws.POST("/v1/namespaces/{namespace}/pods").
		Filter(utils.NamespaceValidation(svc.adapter)).
		To(svc.CreatePod).
		Param(ws.PathParameter("namespace", "namespace name").DataType("string")).
		Param(ws.QueryParameter("dryRun", "when present, indicates that modifications should not be persisted").DataType("string")))

	ws.Route(ws.GET("/v1/pods").
		To(svc.ListPods))

	ws.Route(ws.GET("/v1/namespaces/{namespace}/pods").
		Filter(utils.NamespaceValidation(svc.adapter)).
		To(svc.ListPods).
		Param(ws.PathParameter("namespace", "namespace name").DataType("string")))

	ws.Route(ws.DELETE("/v1/pods/{name}").
		To(svc.DeletePod).
		Param(ws.PathParameter("name", "name of the pod").DataType("string")))

	ws.Route(ws.DELETE("/v1/namespaces/{namespace}/pods/{name}").
		Filter(utils.NamespaceValidation(svc.adapter)).
		To(svc.DeletePod).
		Param(ws.PathParameter("namespace", "namespace name").DataType("string")).
		Param(ws.PathParameter("name", "name of the pod").DataType("string")))

	ws.Route(ws.GET("/v1/pods/{name}").
		To(svc.GetPod).
		Param(ws.PathParameter("name", "name of the pod").DataType("string")))

	ws.Route(ws.GET("/v1/namespaces/{namespace}/pods/{name}").
		Filter(utils.NamespaceValidation(svc.adapter)).
		To(svc.GetPod).
		Param(ws.PathParameter("namespace", "namespace name").DataType("string")).
		Param(ws.PathParameter("name", "name of the pod").DataType("string")))

	ws.Route(ws.PATCH("/v1/pods/{name}").
		To(svc.PatchPod).
		Param(ws.PathParameter("name", "name of the pod").DataType("string")).
		Param(ws.QueryParameter("dryRun", "when present, indicates that modifications should not be persisted").DataType("string")).
		AddExtension("x-kubernetes-group-version-kind", podGVKExtension))

	ws.Route(ws.PATCH("/v1/namespaces/{namespace}/pods/{name}").
		Filter(utils.NamespaceValidation(svc.adapter)).
		To(svc.PatchPod).
		Param(ws.PathParameter("namespace", "namespace name").DataType("string")).
		Param(ws.PathParameter("name", "name of the pod").DataType("string")).
		Param(ws.QueryParameter("dryRun", "when present, indicates that modifications should not be persisted").DataType("string")).
		AddExtension("x-kubernetes-group-version-kind", podGVKExtension))

	ws.Route(ws.GET("/v1/namespaces/{namespace}/pods/{name}/log").
		Filter(utils.NamespaceValidation(svc.adapter)).
		To(svc.GetPodLogs)).
		Param(ws.PathParameter("namespace", "namespace name").DataType("string")).
		Param(ws.PathParameter("name", "name of the pod").DataType("string")).
		Param(ws.QueryParameter("follow", "follow the log stream of the pod").DataType("boolean")).
		Param(ws.QueryParameter("tailLines", "the number of lines from the end of the logs to show").DataType("integer")).
		Param(ws.QueryParameter("timestamps", "add an RFC3339 or RFC3339Nano timestamp at the beginning of every line of log output").DataType("boolean"))
}
