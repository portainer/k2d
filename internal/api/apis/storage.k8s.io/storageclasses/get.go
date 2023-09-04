package storageclasses

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/emicklei/go-restful/v3"

	adaptererr "github.com/portainer/k2d/internal/adapter/errors"
	"github.com/portainer/k2d/internal/api/utils"
)

func (svc StorageClassService) GetStorageClass(r *restful.Request, w *restful.Response) {
	storageClassName := r.PathParameter("name")

	sc, err := svc.adapter.GetStorageClass(r.Request.Context(), storageClassName)
	if err != nil {
		if errors.Is(err, adaptererr.ErrResourceNotFound) {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		utils.HttpError(r, w, http.StatusInternalServerError, fmt.Errorf("unable to get storage class: %w", err))
		return
	}

	w.WriteAsJson(sc)
}
