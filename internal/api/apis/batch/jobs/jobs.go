package jobs

import (
	"github.com/emicklei/go-restful/v3"
	"github.com/portainer/k2d/internal/adapter"
	"github.com/portainer/k2d/internal/controller"
)

type JobService struct {
	adapter    *adapter.KubeDockerAdapter
	operations chan controller.Operation
}

func NewJobService(adapter *adapter.KubeDockerAdapter, operations chan controller.Operation) JobService {
	return JobService{
		adapter:    adapter,
		operations: operations,
	}
}

func (svc JobService) RegisterJobAPI(ws *restful.WebService) {
	jobGVKExtension := map[string]string{
		"group":   "batch",
		"kind":    "Job",
		"version": "v1",
	}

	ws.Route(ws.POST("/v1/jobs").
		To(svc.CreateJob).
		Param(ws.QueryParameter("dryRun", "when present, indicates that modifications should not be persisted").DataType("string")))

	ws.Route(ws.POST("/v1/namespaces/{namespace}/jobs").
		To(svc.CreateJob).
		Param(ws.PathParameter("namespace", "namespace name").DataType("string")).
		Param(ws.QueryParameter("dryRun", "when present, indicates that modifications should not be persisted").DataType("string")))

	ws.Route(ws.GET("/v1/jobs").
		To(svc.ListJobs))

	ws.Route(ws.GET("/v1/namespaces/{namespace}/jobs").
		To(svc.ListJobs).
		Param(ws.PathParameter("namespace", "namespace name").DataType("string")))

	ws.Route(ws.DELETE("/v1/jobs/{name}").
		To(svc.DeleteJob).
		Param(ws.PathParameter("name", "name of the job").DataType("string")))

	ws.Route(ws.DELETE("/v1/namespaces/{namespace}/jobs/{name}").
		To(svc.DeleteJob).
		Param(ws.PathParameter("namespace", "namespace name").DataType("string")).
		Param(ws.PathParameter("name", "name of the job").DataType("string")))

	ws.Route(ws.GET("/v1/jobs/{name}").
		To(svc.GetJob).
		Param(ws.PathParameter("name", "name of the job").DataType("string")))

	ws.Route(ws.GET("/v1/namespaces/{namespace}/jobs/{name}").
		To(svc.GetJob).
		Param(ws.PathParameter("namespace", "namespace name").DataType("string")).
		Param(ws.PathParameter("name", "name of the job").DataType("string")))

	ws.Route(ws.PATCH("/v1/jobs/{name}").
		To(svc.PatchJob).
		Param(ws.PathParameter("name", "name of the job").DataType("string")).
		Param(ws.QueryParameter("dryRun", "when present, indicates that modifications should not be persisted").DataType("string")).
		AddExtension("x-kubernetes-group-version-kind", jobGVKExtension))

	ws.Route(ws.PATCH("/v1/namespaces/{namespace}/jobs/{name}").
		To(svc.PatchJob).
		Param(ws.PathParameter("namespace", "namespace name").DataType("string")).
		Param(ws.PathParameter("name", "name of the job").DataType("string")).
		Param(ws.QueryParameter("dryRun", "when present, indicates that modifications should not be persisted").DataType("string")).
		AddExtension("x-kubernetes-group-version-kind", jobGVKExtension))
}
