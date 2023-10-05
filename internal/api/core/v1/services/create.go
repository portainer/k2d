package services

import (
	"fmt"
	"net/http"

	"github.com/emicklei/go-restful/v3"
	"github.com/portainer/k2d/internal/api/utils"
	"github.com/portainer/k2d/internal/controller"
	"github.com/portainer/k2d/internal/types"
	httputils "github.com/portainer/k2d/pkg/http"
	corev1 "k8s.io/api/core/v1"
)

func (svc ServiceService) CreateService(r *restful.Request, w *restful.Response) {
	namespace := utils.GetNamespaceFromRequest(r)

	service := &corev1.Service{}
	err := httputils.ParseJSONBody(r.Request, &service)
	if err != nil {
		utils.HttpError(r, w, http.StatusBadRequest, fmt.Errorf("unable to parse request body: %w", err))
		return
	}

	service.Namespace = namespace

	dryRun := r.QueryParameter("dryRun") != ""
	if dryRun {
		w.WriteAsJson(service)
		return
	}

	svc.operations <- controller.NewOperation(service, controller.LowPriorityOperation, r.HeaderParameter(types.RequestIDHeader))

	w.WriteAsJson(service)
}
