package secrets

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/emicklei/go-restful/v3"
	adaptererr "github.com/portainer/k2d/internal/adapter/errors"
	"github.com/portainer/k2d/internal/api/utils"
)

func (svc SecretService) GetSecret(r *restful.Request, w *restful.Response) {
	namespace := utils.GetNamespaceFromRequest(r)
	secretName := r.PathParameter("name")

	secret, err := svc.adapter.GetSecret(secretName, namespace)
	if err != nil {
		if errors.Is(err, adaptererr.ErrResourceNotFound) {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		utils.HttpError(r, w, http.StatusInternalServerError, fmt.Errorf("unable to get secret: %w", err))
		return
	}

	w.WriteAsJson(secret)
}
