package metrics

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/emicklei/go-restful/v3"
	adaptererr "github.com/portainer/k2d/internal/adapter/errors"
	"github.com/portainer/k2d/internal/api/utils"
)

func (svc MetricsService) GetMetrics(r *restful.Request, w *restful.Response) {
	namespace := utils.GetNamespaceFromRequest(r)
	podName := r.PathParameter("name")

	podMetrics, err := svc.adapter.GetPodMetrics(r.Request.Context(), namespace, podName)
	if err != nil {
		if errors.Is(err, adaptererr.ErrResourceNotFound) {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		utils.HttpError(r, w, http.StatusInternalServerError, fmt.Errorf("unable to get pod metrics: %w", err))
		return
	}

	w.WriteAsJson(podMetrics)
}
