package jobs

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/emicklei/go-restful/v3"
	"github.com/portainer/k2d/internal/api/utils"
	"github.com/portainer/k2d/internal/controller"
	"github.com/portainer/k2d/internal/types"
	batchv1 "k8s.io/api/batch/v1"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
)

func (svc JobService) PatchJob(r *restful.Request, w *restful.Response) {
	namespace := r.PathParameter("namespace")
	jobName := r.PathParameter("name")

	patch, err := io.ReadAll(r.Request.Body)
	if err != nil {
		utils.HttpError(r, w, http.StatusBadRequest, fmt.Errorf("unable to parse request body: %w", err))
		return
	}

	job, err := svc.adapter.GetJob(r.Request.Context(), jobName, namespace)
	if err != nil {
		utils.HttpError(r, w, http.StatusInternalServerError, fmt.Errorf("unable to get job: %w", err))
		return
	}

	if job == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	data, err := json.Marshal(job)
	if err != nil {
		utils.HttpError(r, w, http.StatusInternalServerError, fmt.Errorf("unable to marshal job: %w", err))
		return
	}

	mergedData, err := strategicpatch.StrategicMergePatch(data, patch, batchv1.Job{})
	if err != nil {
		utils.HttpError(r, w, http.StatusInternalServerError, fmt.Errorf("unable to apply patch: %w", err))
		return
	}

	updatedJob := &batchv1.Job{}

	err = json.Unmarshal(mergedData, updatedJob)
	if err != nil {
		utils.HttpError(r, w, http.StatusInternalServerError, fmt.Errorf("unable to unmarshal job: %w", err))
		return
	}

	dryRun := r.QueryParameter("dryRun") != ""
	if dryRun {
		w.WriteAsJson(updatedJob)
		return
	}

	svc.operations <- controller.NewOperation(updatedJob, controller.MediumPriorityOperation, r.HeaderParameter(types.RequestIDHeader))

	w.WriteAsJson(updatedJob)
}
