package persistentvolumeclaims

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/emicklei/go-restful/v3"
	adaptererr "github.com/portainer/k2d/internal/adapter/errors"
	"github.com/portainer/k2d/internal/api/utils"
)

func (svc PersistentVolumeClaimService) GetPersistentVolumeClaim(r *restful.Request, w *restful.Response) {
	namespace := r.PathParameter("namespace")
	persistentVolumeClaimName := r.PathParameter("name")

	persistentVolumeClaim, err := svc.adapter.GetPersistentVolumeClaim(r.Request.Context(), persistentVolumeClaimName, namespace)
	if err != nil {
		if errors.Is(err, adaptererr.ErrResourceNotFound) {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		utils.HttpError(r, w, http.StatusInternalServerError, fmt.Errorf("unable to get persistent volume claim: %w", err))
		return
	}

	w.WriteAsJson(persistentVolumeClaim)
}
