package nodes

import (
	"github.com/emicklei/go-restful/v3"
	"github.com/portainer/k2d/internal/adapter"
)

type NodeService struct {
	adapter *adapter.KubeDockerAdapter
}

func NewNodeService(adapter *adapter.KubeDockerAdapter) NodeService {
	return NodeService{
		adapter: adapter,
	}
}

func (svc NodeService) RegisterNodeAPI(ws *restful.WebService) {
	ws.Route(ws.GET("/v1/nodes").
		To(svc.ListNodes))
}
