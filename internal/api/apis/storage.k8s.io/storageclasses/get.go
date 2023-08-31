package storageclasses

import (
	"fmt"
	"net/http"

	"github.com/emicklei/go-restful/v3"
	"github.com/portainer/k2d/internal/api/utils"
)

func (svc StorageClassService) GetStorageClass(r *restful.Request, w *restful.Response) {
	storageClassName := r.PathParameter("name")

	sc, err := svc.adapter.GetStorageClass(r.Request.Context(), storageClassName)
	if err != nil {
		utils.HttpError(r, w, http.StatusInternalServerError, fmt.Errorf("unable to get the storage class: %w", err))
		return
	}

	if sc == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	w.WriteAsJson(sc)
}
