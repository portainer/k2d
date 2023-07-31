package k2d

import (
	"fmt"

	"github.com/emicklei/go-restful/v3"
	"github.com/portainer/k2d/internal/adapter"
	"github.com/portainer/k2d/internal/api/k2d/config"
	"github.com/portainer/k2d/internal/api/k2d/system"
	"github.com/portainer/k2d/internal/types"
)

type (
	K2DAPI struct {
		configService config.ConfigService
		systemService system.SystemService
	}
)

func NewK2DAPI(cfg *types.K2DServerConfiguration, adapter *adapter.KubeDockerAdapter) *K2DAPI {
	serverAddress := fmt.Sprintf("https://%s:%d", cfg.ServerIpAddr, cfg.ServerPort)

	return &K2DAPI{
		configService: config.NewConfigService(cfg.CaPath, serverAddress, cfg.Secret),
		systemService: system.NewSystemService(cfg, adapter),
	}
}

// /k2d/kubeconfig
func (api K2DAPI) Kubeconfig() *restful.WebService {
	routes := new(restful.WebService).
		Path("/k2d/kubeconfig").
		Produces("application/yml")

	routes.Route(routes.GET("").
		To(api.configService.GetKubeconfig))

	return routes
}

func (api K2DAPI) System() *restful.WebService {
	routes := new(restful.WebService).
		Path("/k2d/system").
		Produces(restful.MIME_JSON)

	routes.Route(routes.GET("/diagnostics").
		To(api.systemService.Diagnostics))

	return routes
}
