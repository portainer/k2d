package metrics

import (
	"errors"
	"net/http"

	"github.com/emicklei/go-restful/v3"
	adaptererr "github.com/portainer/k2d/internal/adapter/errors"
)

func (svc MetricsService) GetMetrics(r *restful.Request, w *restful.Response) {
	namespace := r.PathParameter("namespace")
	podName := r.PathParameter("name")

	podMetrics, err := svc.adapter.GetPodMetrics(r.Request.Context(), podName, namespace)
	if err != nil {
		if errors.Is(err, adaptererr.ErrResourceNotFound) {
			w.WriteHeader(http.StatusNotFound)
			return
		}
	}

	w.WriteAsJson(podMetrics)
}
