package jobs

import (
	"fmt"
	"net/http"

	"github.com/emicklei/go-restful/v3"
	"github.com/portainer/k2d/internal/api/utils"
	"github.com/portainer/k2d/internal/controller"
	"github.com/portainer/k2d/internal/types"
	httputils "github.com/portainer/k2d/pkg/http"
	batchv1 "k8s.io/api/batch/v1"
)

func (svc JobService) CreateJob(r *restful.Request, w *restful.Response) {
	namespace := r.PathParameter("namespace")

	job := &batchv1.Job{}

	err := httputils.ParseJSONBody(r.Request, &job)
	if err != nil {
		utils.HttpError(r, w, http.StatusBadRequest, fmt.Errorf("unable to parse request body: %w", err))
		return
	}

	job.Namespace = namespace

	dryRun := r.QueryParameter("dryRun") != ""
	if dryRun {
		w.WriteAsJson(job)
		return
	}

	svc.operations <- controller.NewOperation(job, controller.MediumPriorityOperation, r.HeaderParameter(types.RequestIDHeader))

	w.WriteAsJson(job)
}
