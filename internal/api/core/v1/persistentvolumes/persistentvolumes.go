package persistentvolumes

import (
	"github.com/emicklei/go-restful/v3"
	"github.com/portainer/k2d/internal/adapter"
)

type PersistentVolumeService struct {
	adapter *adapter.KubeDockerAdapter
}

func NewPersistentVolumeService(adapter *adapter.KubeDockerAdapter) PersistentVolumeService {
	return PersistentVolumeService{
		adapter: adapter,
	}
}

func (svc PersistentVolumeService) RegisterPersistentVolumeAPI(ws *restful.WebService) {
	ws.Route(ws.GET("/v1/persistentvolumes").
		To(svc.ListPersistentVolumes))

	ws.Route(ws.GET("/v1/persistentvolumes/{name}").
		To(svc.GetPersistentVolume).
		Param(ws.PathParameter("name", "name of the persistentvolume").DataType("string")))

	ws.Route(ws.DELETE("/v1/persistentvolumes/{name}").
		To(svc.GetPersistentVolume).
		Param(ws.PathParameter("name", "name of the persistentvolumes").DataType("string")))
}
