package persistentvolumeclaims

import (
	"github.com/emicklei/go-restful/v3"
	"github.com/portainer/k2d/internal/adapter"
	"github.com/portainer/k2d/internal/api/utils"
	"github.com/portainer/k2d/internal/controller"
)

type PersistentVolumeClaimService struct {
	adapter    *adapter.KubeDockerAdapter
	operations chan controller.Operation
}

func NewPersistentVolumeClaimService(adapter *adapter.KubeDockerAdapter, operations chan controller.Operation) PersistentVolumeClaimService {
	return PersistentVolumeClaimService{
		adapter:    adapter,
		operations: operations,
	}
}

func (svc PersistentVolumeClaimService) RegisterPersistentVolumeClaimAPI(ws *restful.WebService) {
	persistentVolumeClaimGVKExtension := map[string]string{
		"group":   "",
		"kind":    "PersistentVolumeClaim",
		"version": "v1",
	}

	ws.Route(ws.POST("/v1/persistentvolumeclaims").
		To(svc.CreatePersistentVolumeClaim).
		Param(ws.QueryParameter("dryRun", "when present, indicates that modifications should not be persisted").DataType("string")))

	ws.Route(ws.POST("/v1/namespaces/{namespace}/persistentvolumeclaims").
		Filter(utils.NamespaceValidation(svc.adapter)).
		To(svc.CreatePersistentVolumeClaim).
		Param(ws.PathParameter("namespace", "namespace name").DataType("string")).
		Param(ws.QueryParameter("dryRun", "when present, indicates that modifications should not be persisted").DataType("string")))

	ws.Route(ws.GET("/v1/persistentvolumeclaims").
		To(svc.ListPersistentVolumeClaims))

	ws.Route(ws.GET("/v1/namespaces/{namespace}/persistentvolumeclaims").
		Filter(utils.NamespaceValidation(svc.adapter)).
		To(svc.ListPersistentVolumeClaims).
		Param(ws.PathParameter("namespace", "namespace name").DataType("string")))

	ws.Route(ws.GET("/v1/persistentvolumeclaims/{name}").
		To(svc.GetPersistentVolumeClaim).
		Param(ws.PathParameter("name", "name of the persistentvolumeclaims").DataType("string")))

	ws.Route(ws.GET("/v1/namespaces/{namespace}/persistentvolumeclaims/{name}").
		Filter(utils.NamespaceValidation(svc.adapter)).
		To(svc.GetPersistentVolumeClaim).
		Param(ws.PathParameter("namespace", "namespace name").DataType("string")).
		Param(ws.PathParameter("name", "name of the persistentvolumeclaim").DataType("string")))

	ws.Route(ws.DELETE("/v1/persistentvolumeclaims/{name}").
		To(svc.DeletePersistentVolumeClaim).
		Param(ws.PathParameter("name", "name of the persistentvolumeclaim").DataType("string")))

	ws.Route(ws.DELETE("/v1/namespaces/{namespace}/persistentvolumeclaims/{name}").
		Filter(utils.NamespaceValidation(svc.adapter)).
		To(svc.DeletePersistentVolumeClaim).
		Param(ws.PathParameter("namespace", "namespace name").DataType("string")).
		Param(ws.PathParameter("name", "name of the persistentvolumeclaim").DataType("string")))

	ws.Route(ws.PATCH("/v1/persistentvolumeclaims/{name}").
		To(svc.PatchPersistentVolumeClaim).
		Param(ws.PathParameter("name", "name of the persistentvolumeclaim").DataType("string")).
		Param(ws.QueryParameter("dryRun", "when present, indicates that modifications should not be persisted").DataType("string")).
		AddExtension("x-kubernetes-group-version-kind", persistentVolumeClaimGVKExtension))

	ws.Route(ws.PATCH("/v1/namespaces/{namespace}/persistentvolumeclaims/{name}").
		Filter(utils.NamespaceValidation(svc.adapter)).
		To(svc.PatchPersistentVolumeClaim).
		Param(ws.PathParameter("namespace", "namespace name").DataType("string")).
		Param(ws.PathParameter("name", "name of the persistentvolumeclaim").DataType("string")).
		Param(ws.QueryParameter("dryRun", "when present, indicates that modifications should not be persisted").DataType("string")).
		AddExtension("x-kubernetes-group-version-kind", persistentVolumeClaimGVKExtension))
}
