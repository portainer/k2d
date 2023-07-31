package deployments

import (
	"fmt"
	"net/http"

	"github.com/emicklei/go-restful/v3"
	"github.com/portainer/k2d/internal/api/utils"
)

func (svc DeploymentService) GetDeployment(r *restful.Request, w *restful.Response) {
	deploymentName := r.PathParameter("name")

	deployment, err := svc.adapter.GetDeployment(r.Request.Context(), deploymentName)
	if err != nil {
		utils.HttpError(r, w, http.StatusInternalServerError, fmt.Errorf("unable to get deployment: %w", err))
		return
	}

	if deployment == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	w.WriteAsJson(deployment)
}
