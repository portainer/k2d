package statefulsets

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

func (svc StatefulSetService) CreateStatefulSet(r *restful.Request, w *restful.Response) {
	statefulSet := &appsv1.StatefulSet{}

	err := httputils.ParseJSONBody(r.Request, &statefulSet)
	if err != nil {
		utils.HttpError(r, w, http.StatusBadRequest, fmt.Errorf("unable to parse request body: %w", err))
		return
	}

	dryRun := r.QueryParameter("dryRun") != ""
	if dryRun {
		w.WriteAsJson(statefulSet)
		return
	}

	svc.operations <- controller.NewOperation(statefulSet, controller.MediumPriorityOperation, r.HeaderParameter(types.RequestIDHeader))

	w.WriteAsJson(statefulSet)
}
