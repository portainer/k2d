package deployments

import (
	"fmt"
	"net/http"

	"github.com/emicklei/go-restful/v3"
	"github.com/portainer/k2d/internal/api/utils"
	"github.com/portainer/k2d/internal/controller"
	"github.com/portainer/k2d/internal/types"
	httputils "github.com/portainer/k2d/pkg/http"
	appsv1 "k8s.io/api/apps/v1"
)

func (svc DeploymentService) CreateDeployment(r *restful.Request, w *restful.Response) {
	namespace := r.PathParameter("namespace")

	deployment := &appsv1.Deployment{}

	err := httputils.ParseJSONBody(r.Request, &deployment)
	if err != nil {
		utils.HttpError(r, w, http.StatusBadRequest, fmt.Errorf("unable to parse request body: %w", err))
		return
	}

	if namespace != "" {
		deployment.Namespace = namespace
	}

	dryRun := r.QueryParameter("dryRun") != ""
	if dryRun {
		w.WriteAsJson(deployment)
		return
	}

	svc.operations <- controller.NewOperation(deployment, controller.MediumPriorityOperation, r.HeaderParameter(types.RequestIDHeader))

	w.WriteAsJson(deployment)
}
