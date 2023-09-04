package jobs

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/emicklei/go-restful/v3"
	adaptererr "github.com/portainer/k2d/internal/adapter/errors"
	"github.com/portainer/k2d/internal/api/utils"
)

func (svc JobService) GetJob(r *restful.Request, w *restful.Response) {
	namespace := r.PathParameter("namespace")
	jobName := r.PathParameter("name")

	job, err := svc.adapter.GetJob(r.Request.Context(), jobName, namespace)
	if err != nil {
		if errors.Is(err, adaptererr.ErrResourceNotFound) {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		utils.HttpError(r, w, http.StatusInternalServerError, fmt.Errorf("unable to get job: %w", err))
		return
	}

	w.WriteAsJson(job)
}
