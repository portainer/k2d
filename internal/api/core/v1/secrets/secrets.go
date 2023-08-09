package secrets

import (
	"github.com/emicklei/go-restful/v3"
	"github.com/portainer/k2d/internal/adapter"
	"github.com/portainer/k2d/internal/controller"
)

type SecretService struct {
	adapter    *adapter.KubeDockerAdapter
	operations chan controller.Operation
}

func NewSecretService(adapter *adapter.KubeDockerAdapter, operations chan controller.Operation) SecretService {
	return SecretService{
		adapter:    adapter,
		operations: operations,
	}
}

func (svc SecretService) RegisterSecretAPI(ws *restful.WebService) {
	secretGVKExtension := map[string]string{
		"group":   "",
		"kind":    "Secret",
		"version": "v1",
	}

	ws.Route(ws.POST("/v1/secrets").
		To(svc.CreateSecret).
		Param(ws.QueryParameter("dryRun", "when present, indicates that modifications should not be persisted").DataType("string")))

	ws.Route(ws.POST("/v1/namespaces/{namespace}/secrets").
		To(svc.CreateSecret).
		Param(ws.QueryParameter("dryRun", "when present, indicates that modifications should not be persisted").DataType("string")))

	ws.Route(ws.GET("/v1/secrets").
		Param(ws.QueryParameter("labelSelector", "a selector to restrict the list of returned objects by their labels").DataType("string")).
		To(svc.ListSecrets))

	ws.Route(ws.GET("/v1/namespaces/{namespace}/secrets").
		Param(ws.QueryParameter("labelSelector", "a selector to restrict the list of returned objects by their labels").DataType("string")).
		To(svc.ListSecrets))

	ws.Route(ws.DELETE("/v1/secrets/{name}").
		To(svc.DeleteSecret).
		Param(ws.PathParameter("name", "name of the secret").DataType("string")))

	ws.Route(ws.DELETE("/v1/namespaces/{namespace}/secrets/{name}").
		To(svc.DeleteSecret).
		Param(ws.PathParameter("name", "name of the secret").DataType("string")))

	ws.Route(ws.GET("/v1/secrets/{name}").
		To(svc.GetSecret).
		Param(ws.PathParameter("name", "name of the secret").DataType("string")))

	ws.Route(ws.GET("/v1/namespaces/{namespace}/secrets/{name}").
		To(svc.GetSecret).
		Param(ws.PathParameter("name", "name of the secret").DataType("string")))

	ws.Route(ws.PATCH("/v1/secrets/{name}").
		To(svc.PatchSecret).
		Param(ws.QueryParameter("dryRun", "when present, indicates that modifications should not be persisted").DataType("string")).
		AddExtension("x-kubernetes-group-version-kind", secretGVKExtension).
		Param(ws.PathParameter("name", "name of the secret").DataType("string")))

	ws.Route(ws.PATCH("/v1/namespaces/{namespace}/secrets/{name}").
		To(svc.PatchSecret).
		Param(ws.QueryParameter("dryRun", "when present, indicates that modifications should not be persisted").DataType("string")).
		AddExtension("x-kubernetes-group-version-kind", secretGVKExtension).
		Param(ws.PathParameter("name", "name of the secret").DataType("string")))

	ws.Route(ws.PUT("/v1/secrets/{name}").
		To(svc.PutSecret).
		Param(ws.QueryParameter("dryRun", "when present, indicates that modifications should not be persisted").DataType("string")).
		AddExtension("x-kubernetes-group-version-kind", secretGVKExtension).
		Param(ws.PathParameter("name", "name of the secret").DataType("string")))

	ws.Route(ws.PUT("/v1/namespaces/{namespace}/secrets/{name}").
		To(svc.PutSecret).
		Param(ws.QueryParameter("dryRun", "when present, indicates that modifications should not be persisted").DataType("string")).
		AddExtension("x-kubernetes-group-version-kind", secretGVKExtension).
		Param(ws.PathParameter("name", "name of the secret").DataType("string")))
}
