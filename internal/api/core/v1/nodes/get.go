package nodes

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/emicklei/go-restful/v3"
	adaptererr "github.com/portainer/k2d/internal/adapter/errors"

	"github.com/portainer/k2d/internal/api/utils"
)

func (svc NodeService) GetNode(r *restful.Request, w *restful.Response) {
	name := r.PathParameter("name")

	node, err := svc.adapter.GetNode(r.Request.Context(), name)
	if err != nil {
		if errors.Is(err, adaptererr.ErrResourceNotFound) {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		utils.HttpError(r, w, http.StatusInternalServerError, fmt.Errorf("unable to get namespace: %w", err))
		return
	}

	w.WriteAsJson(node)
}
