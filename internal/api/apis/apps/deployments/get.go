package deployments

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/emicklei/go-restful/v3"
	adaptererr "github.com/portainer/k2d/internal/adapter/errors"
	"github.com/portainer/k2d/internal/api/utils"
)

func (svc DeploymentService) GetDeployment(r *restful.Request, w *restful.Response) {
	namespace := utils.NamespaceParameter(r)
	deploymentName := r.PathParameter("name")

	deployment, err := svc.adapter.GetDeployment(r.Request.Context(), deploymentName, namespace)
	if err != nil {
		if errors.Is(err, adaptererr.ErrResourceNotFound) {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		utils.HttpError(r, w, http.StatusInternalServerError, fmt.Errorf("unable to get deployment: %w", err))
		return
	}

	w.WriteAsJson(deployment)
}
