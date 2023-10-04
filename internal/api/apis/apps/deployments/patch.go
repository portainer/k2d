package deployments

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/emicklei/go-restful/v3"
	"github.com/portainer/k2d/internal/api/utils"
	"github.com/portainer/k2d/internal/controller"
	"github.com/portainer/k2d/internal/types"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
)

func (svc DeploymentService) PatchDeployment(r *restful.Request, w *restful.Response) {
	namespace := r.PathParameter("namespace")
	// namespace validation. if doesn't exist, return 404
	utils.ValidateNamespace(r, w, svc.adapter, namespace)

	deploymentName := r.PathParameter("name")
	patch, err := io.ReadAll(r.Request.Body)
	if err != nil {
		utils.HttpError(r, w, http.StatusBadRequest, fmt.Errorf("unable to parse request body: %w", err))
		return
	}

	deployment, err := svc.adapter.GetDeployment(r.Request.Context(), deploymentName, namespace)
	if err != nil {
		utils.HttpError(r, w, http.StatusInternalServerError, fmt.Errorf("unable to get deployment: %w", err))
		return
	}

	if deployment == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	data, err := json.Marshal(deployment)
	if err != nil {
		utils.HttpError(r, w, http.StatusInternalServerError, fmt.Errorf("unable to marshal deployment: %w", err))
		return
	}

	mergedData, err := strategicpatch.StrategicMergePatch(data, patch, appsv1.Deployment{})
	if err != nil {
		utils.HttpError(r, w, http.StatusInternalServerError, fmt.Errorf("unable to apply patch: %w", err))
		return
	}

	updatedDeployment := &appsv1.Deployment{}

	err = json.Unmarshal(mergedData, updatedDeployment)
	if err != nil {
		utils.HttpError(r, w, http.StatusInternalServerError, fmt.Errorf("unable to unmarshal deployment: %w", err))
		return
	}

	dryRun := r.QueryParameter("dryRun") != ""
	if dryRun {
		w.WriteAsJson(updatedDeployment)
		return
	}

	svc.operations <- controller.NewOperation(updatedDeployment, controller.MediumPriorityOperation, r.HeaderParameter(types.RequestIDHeader))

	w.WriteAsJson(updatedDeployment)
}
