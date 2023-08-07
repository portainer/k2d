package nodes

import (
	"fmt"
	"net/http"

	"github.com/emicklei/go-restful/v3"
	"github.com/portainer/k2d/internal/api/utils"
)

func (svc NodeService) GetNode(r *restful.Request, w *restful.Response) {
	node, err := svc.adapter.GetNode(r.Request.Context())
	if err != nil {
		utils.HttpError(r, w, http.StatusInternalServerError, fmt.Errorf("unable to get node: %w", err))
		return
	}

	if node == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	w.WriteAsJson(node)
}
