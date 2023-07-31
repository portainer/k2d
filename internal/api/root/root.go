package root

import (
	"github.com/emicklei/go-restful/v3"
	"github.com/portainer/k2d/internal/api/root/healthz"
	"github.com/portainer/k2d/internal/api/root/version"
)

type (
	Root struct {
		version version.VersionService
		health  healthz.HealthzService
	}
)

func NewRootAPI() *Root {
	return &Root{
		version: version.NewVersionService(),
		health:  healthz.NewHealthzService(),
	}
}

// /healthz
func (api Root) Healthz() *restful.WebService {
	routes := new(restful.WebService).
		Path("/healthz")

	routes.Route(routes.GET("").
		To(api.health.Healthz))

	return routes
}

// /version
func (api Root) Version() *restful.WebService {
	routes := new(restful.WebService).
		Path("/version").
		Consumes(restful.MIME_JSON, "application/yml").
		Produces(restful.MIME_JSON, "application/yml")

	routes.Route(routes.GET("").
		To(api.version.Version))

	return routes
}
