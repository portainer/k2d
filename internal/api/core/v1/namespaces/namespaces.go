package namespaces

import (
	"github.com/emicklei/go-restful/v3"
	"github.com/portainer/k2d/internal/adapter"
)

type NamespaceService struct {
	adapter *adapter.KubeDockerAdapter
}

func NewNamespaceService(adapter *adapter.KubeDockerAdapter) NamespaceService {
	return NamespaceService{
		adapter: adapter,
	}
}

func (svc NamespaceService) RegisterNamespaceAPI(ws *restful.WebService) {
	ws.Route(ws.POST("/v1/namespaces").
		To(svc.CreateNamespace))

	ws.Route(ws.GET("/v1/namespaces").
		To(svc.ListNamespaces))
}
