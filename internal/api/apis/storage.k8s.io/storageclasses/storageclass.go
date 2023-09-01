package storageclasses

import (
	"github.com/emicklei/go-restful/v3"
	"github.com/portainer/k2d/internal/adapter"
)

type StorageClassService struct {
	adapter *adapter.KubeDockerAdapter
}

func NewStorageClassService(adapter *adapter.KubeDockerAdapter) StorageClassService {
	return StorageClassService{
		adapter: adapter,
	}
}

func (svc StorageClassService) RegisterStorageClassAPI(ws *restful.WebService) {
	ws.Route(ws.GET("/v1/storageclasses").
		To(svc.ListStorageClass))

	ws.Route(ws.GET("/v1/storageclasses/{name}").
		To(svc.GetStorageClass).
		Param(ws.PathParameter("name", "name of the storageclass").DataType("string")))
}
