package namespaces

import (
	"fmt"
	"net/http"

	"github.com/emicklei/go-restful/v3"
	"github.com/portainer/k2d/internal/api/utils"
)

func (svc NamespaceService) GetNamespace(r *restful.Request, w *restful.Response) {
	name := r.PathParameter("name")
	watch := r.PathParameter("watch")

	namespace, err := svc.adapter.GetNamespace(r.Request.Context(), name, watch)
	if err != nil {
		utils.HttpError(r, w, http.StatusInternalServerError, fmt.Errorf("unable to get namespace: %w", err))
		return
	}

	if namespace == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	w.WriteAsJson(namespace)
}
