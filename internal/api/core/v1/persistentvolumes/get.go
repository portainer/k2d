package persistentvolumes

import (
	"fmt"
	"net/http"

	"github.com/emicklei/go-restful/v3"
	"github.com/portainer/k2d/internal/api/utils"
)

func (svc PersistentVolumeService) GetPersistentVolume(r *restful.Request, w *restful.Response) {
	persistentVolumeName := r.PathParameter("name")

	persistentVolume, err := svc.adapter.GetPersistentVolume(r.Request.Context(), persistentVolumeName)
	if err != nil {
		utils.HttpError(r, w, http.StatusInternalServerError, fmt.Errorf("unable to get persistentVolume: %w", err))
		return
	}

	if persistentVolume == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	w.WriteAsJson(persistentVolume)
}
