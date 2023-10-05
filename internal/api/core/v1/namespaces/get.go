package namespaces

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/emicklei/go-restful/v3"
	adaptererr "github.com/portainer/k2d/internal/adapter/errors"
	"github.com/portainer/k2d/internal/api/utils"
)

func (svc NamespaceService) GetNamespace(r *restful.Request, w *restful.Response) {
	namespaceName := utils.GetNamespaceFromRequest(r)

	namespace, err := svc.adapter.GetNamespace(r.Request.Context(), namespaceName)
	if err != nil {
		if errors.Is(err, adaptererr.ErrResourceNotFound) {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		utils.HttpError(r, w, http.StatusInternalServerError, fmt.Errorf("unable to get namespace: %w", err))
		return
	}

	w.WriteAsJson(namespace)
}
