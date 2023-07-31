package daemonsets

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

func (svc DaemonSetService) CreateDaemonSet(r *restful.Request, w *restful.Response) {
	daemonset := &appsv1.DaemonSet{}

	err := httputils.ParseJSONBody(r.Request, &daemonset)
	if err != nil {
		utils.HttpError(r, w, http.StatusBadRequest, fmt.Errorf("unable to parse request body: %w", err))
		return
	}

	dryRun := r.QueryParameter("dryRun") != ""
	if dryRun {
		w.WriteAsJson(daemonset)
		return
	}

	svc.operations <- controller.NewOperation(daemonset, controller.MediumPriorityOperation, r.HeaderParameter(types.RequestIDHeader))

	w.WriteAsJson(daemonset)
}
