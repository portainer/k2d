package persistentvolumes

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/emicklei/go-restful/v3"
	adaptererr "github.com/portainer/k2d/internal/adapter/errors"
	"github.com/portainer/k2d/internal/api/utils"
)

func (svc PersistentVolumeService) GetPersistentVolume(r *restful.Request, w *restful.Response) {
	persistentVolumeName := r.PathParameter("name")

	persistentVolume, err := svc.adapter.GetPersistentVolume(r.Request.Context(), persistentVolumeName)
	if err != nil {
		if errors.Is(err, adaptererr.ErrResourceNotFound) {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		utils.HttpError(r, w, http.StatusInternalServerError, fmt.Errorf("unable to get persistent volume: %w", err))
		return
	}

	w.WriteAsJson(persistentVolume)
}
