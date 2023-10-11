package deployments

import (
	"github.com/emicklei/go-restful/v3"
	"github.com/portainer/k2d/internal/adapter"
	"github.com/portainer/k2d/internal/api/utils"
	"github.com/portainer/k2d/internal/controller"
)

type DeploymentService struct {
	adapter    *adapter.KubeDockerAdapter
	operations chan controller.Operation
}

func NewDeploymentService(adapter *adapter.KubeDockerAdapter, operations chan controller.Operation) DeploymentService {
	return DeploymentService{
		adapter:    adapter,
		operations: operations,
	}
}

func (svc DeploymentService) RegisterDeploymentAPI(ws *restful.WebService) {
	deploymentGVKExtension := map[string]string{
		"group":   "apps",
		"kind":    "Deployment",
		"version": "v1",
	}

	ws.Route(ws.POST("/v1/deployments").
		To(svc.CreateDeployment).
		Param(ws.QueryParameter("dryRun", "when present, indicates that modifications should not be persisted").DataType("string")))

	ws.Route(ws.POST("/v1/namespaces/{namespace}/deployments").
		Filter(utils.NamespaceValidation(svc.adapter)).
		To(svc.CreateDeployment).
		Param(ws.PathParameter("namespace", "namespace name").DataType("string")).
		Param(ws.QueryParameter("dryRun", "when present, indicates that modifications should not be persisted").DataType("string")))

	ws.Route(ws.GET("/v1/deployments").
		To(svc.ListDeployments))

	ws.Route(ws.GET("/v1/namespaces/{namespace}/deployments").
		Filter(utils.NamespaceValidation(svc.adapter)).
		To(svc.ListDeployments).
		Param(ws.PathParameter("namespace", "namespace name").DataType("string")))

	ws.Route(ws.DELETE("/v1/deployments/{name}").
		To(svc.DeleteDeployment).
		Param(ws.PathParameter("name", "name of the deployment").DataType("string")))

	ws.Route(ws.DELETE("/v1/namespaces/{namespace}/deployments/{name}").
		To(svc.DeleteDeployment).
		Param(ws.PathParameter("namespace", "namespace name").DataType("string")).
		Param(ws.PathParameter("name", "name of the deployment").DataType("string")))

	ws.Route(ws.GET("/v1/deployments/{name}").
		To(svc.GetDeployment).
		Param(ws.PathParameter("name", "name of the deployment").DataType("string")))

	ws.Route(ws.GET("/v1/namespaces/{namespace}/deployments/{name}").
		Filter(utils.NamespaceValidation(svc.adapter)).
		To(svc.GetDeployment).
		Param(ws.PathParameter("namespace", "namespace name").DataType("string")).
		Param(ws.PathParameter("name", "name of the deployment").DataType("string")))

	ws.Route(ws.PATCH("/v1/deployments/{name}").
		To(svc.PatchDeployment).
		Param(ws.PathParameter("name", "name of the deployment").DataType("string")).
		Param(ws.QueryParameter("dryRun", "when present, indicates that modifications should not be persisted").DataType("string")).
		AddExtension("x-kubernetes-group-version-kind", deploymentGVKExtension))

	ws.Route(ws.PATCH("/v1/namespaces/{namespace}/deployments/{name}").
		Filter(utils.NamespaceValidation(svc.adapter)).
		To(svc.PatchDeployment).
		Param(ws.PathParameter("namespace", "namespace name").DataType("string")).
		Param(ws.PathParameter("name", "name of the deployment").DataType("string")).
		Param(ws.QueryParameter("dryRun", "when present, indicates that modifications should not be persisted").DataType("string")).
		AddExtension("x-kubernetes-group-version-kind", deploymentGVKExtension))
}
