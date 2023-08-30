package namespaces

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/emicklei/go-restful/v3"
	"github.com/portainer/k2d/internal/adapter"
	"github.com/portainer/k2d/internal/api/utils"
)

func (svc NamespaceService) GetNamespace(r *restful.Request, w *restful.Response) {
	name := r.PathParameter("name")

	namespace, err := svc.adapter.GetNamespace(r.Request.Context(), name)
	if err != nil {
		if errors.Is(err, adapter.ErrNetworkNotFound) {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		utils.HttpError(r, w, http.StatusInternalServerError, fmt.Errorf("unable to get namespace: %w", err))
		return
	}

	if namespace == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	w.WriteAsJson(namespace)
}
