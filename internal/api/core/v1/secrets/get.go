package secrets

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/emicklei/go-restful/v3"
	storeerr "github.com/portainer/k2d/internal/adapter/store/errors"
	"github.com/portainer/k2d/internal/api/utils"
)

func (svc SecretService) GetSecret(r *restful.Request, w *restful.Response) {
	secretName := r.PathParameter("name")

	secret, err := svc.adapter.GetSecret(secretName)
	if err != nil && errors.Is(err, storeerr.ErrResourceNotFound) {
		w.WriteHeader(http.StatusNotFound)
		return
	} else if err != nil {
		utils.HttpError(r, w, http.StatusInternalServerError, fmt.Errorf("unable to get secret: %w", err))
		return
	}

	w.WriteAsJson(secret)
}
