package core

import (
	restful "github.com/emicklei/go-restful/v3"
	"github.com/portainer/k2d/internal/adapter"
	v1 "github.com/portainer/k2d/internal/api/core/v1"
	"github.com/portainer/k2d/internal/controller"
)

type (
	Core struct {
		v1 v1.V1Service
	}
)

func NewCoreAPI(adapter *adapter.KubeDockerAdapter, operations chan controller.Operation) *Core {
	return &Core{
		v1: v1.NewV1Service(adapter, operations),
	}
}

// /api
func (api Core) V1() *restful.WebService {
	routes := new(restful.WebService).
		Path("/api").
		Consumes(restful.MIME_JSON, "application/yml", "application/json-patch+json", "application/merge-patch+json", "application/strategic-merge-patch+json").
		Produces(restful.MIME_JSON, "application/yml")

	// which versions are served by this api
	routes.Route(routes.GET("").
		To(api.v1.GetAPIVersions))

	// which resources are available under /api/v1
	routes.Route(routes.GET("/v1").
		To(api.v1.ListAPIResources))

	api.v1.RegisterV1API(routes)
	return routes
}
