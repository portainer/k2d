package pods

import (
	"github.com/emicklei/go-restful/v3"
	"github.com/portainer/k2d/internal/adapter"
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
	// This is required for kubectl to be able to use the --dry-run=server flag
	podGVKExtension := map[string]string{
		"group":   "",
		"kind":    "Pod",
		"version": "v1",
	}

	ws.Route(ws.POST("/v1/pods").
		To(svc.CreatePod).
		Param(ws.QueryParameter("dryRun", "when present, indicates that modifications should not be persisted").DataType("string")))

	ws.Route(ws.POST("/v1/namespaces/{namespace}/pods").
		To(svc.CreatePod).
		Param(ws.QueryParameter("dryRun", "when present, indicates that modifications should not be persisted").DataType("string")))

	ws.Route(ws.GET("/v1/pods").
		To(svc.ListPods))

	ws.Route(ws.GET("/v1/namespaces/{namespace}/pods").
		To(svc.ListPods))

	ws.Route(ws.DELETE("/v1/pods/{name}").
		To(svc.DeletePod).
		Param(ws.PathParameter("name", "name of the pod").DataType("string")))

	ws.Route(ws.DELETE("/v1/namespaces/{namespace}/pods/{name}").
		To(svc.DeletePod).
		Param(ws.PathParameter("name", "name of the pod").DataType("string")))

	ws.Route(ws.GET("/v1/pods/{name}").
		To(svc.GetPod).
		Param(ws.PathParameter("name", "name of the pod").DataType("string")))

	ws.Route(ws.GET("/v1/namespaces/{namespace}/pods/{name}").
		To(svc.GetPod).
		Param(ws.PathParameter("name", "name of the pod").DataType("string")))

	ws.Route(ws.PATCH("/v1/pods/{name}").
		To(svc.PatchPod).
		Param(ws.QueryParameter("dryRun", "when present, indicates that modifications should not be persisted").DataType("string")).
		AddExtension("x-kubernetes-group-version-kind", podGVKExtension).
		Param(ws.PathParameter("name", "name of the pod").DataType("string")))

	ws.Route(ws.PATCH("/v1/namespaces/{namespace}/pods/{name}").
		To(svc.PatchPod).
		Param(ws.QueryParameter("dryRun", "when present, indicates that modifications should not be persisted").DataType("string")).
		AddExtension("x-kubernetes-group-version-kind", podGVKExtension).
		Param(ws.PathParameter("name", "name of the pod").DataType("string")))

	ws.Route(ws.GET("/v1/namespaces/{namespace}/pods/{name}/log").
		To(svc.GetPodLogs)).
		Param(ws.PathParameter("name", "name of the pod").DataType("string")).
		Param(ws.QueryParameter("follow", "follow the log stream of the pod").DataType("boolean")).
		Param(ws.QueryParameter("tailLines", "the number of lines from the end of the logs to show").DataType("integer")).
		Param(ws.QueryParameter("timestamps", "add an RFC3339 or RFC3339Nano timestamp at the beginning of every line of log output").DataType("boolean"))
}
