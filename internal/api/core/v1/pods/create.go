package pods

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

func (svc PodService) CreatePod(r *restful.Request, w *restful.Response) {
	namespace := utils.GetNamespaceFromRequest(r)

	pod := &corev1.Pod{}
	err := httputils.ParseJSONBody(r.Request, &pod)
	if err != nil {
		utils.HttpError(r, w, http.StatusBadRequest, fmt.Errorf("unable to parse request body: %w", err))
		return
	}

	pod.Namespace = namespace

	dryRun := r.QueryParameter("dryRun") != ""
	if dryRun {
		w.WriteAsJson(pod)
		return
	}

	svc.operations <- controller.NewOperation(pod, controller.MediumPriorityOperation, r.HeaderParameter(types.RequestIDHeader))

	w.WriteAsJson(pod)
}
