package persistentvolumeclaims

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

func (svc PersistentVolumeClaimService) CreatePersistentVolumeClaim(r *restful.Request, w *restful.Response) {
	namespace := utils.NamespaceParameter(r)
	persistentVolumeClaim := &corev1.PersistentVolumeClaim{}

	err := httputils.ParseJSONBody(r.Request, &persistentVolumeClaim)
	if err != nil {
		utils.HttpError(r, w, http.StatusBadRequest, fmt.Errorf("unable to parse request body: %w", err))
		return
	}

	persistentVolumeClaim.Namespace = namespace

	dryRun := r.QueryParameter("dryRun") != ""
	if dryRun {
		w.WriteAsJson(persistentVolumeClaim)
		return
	}

	svc.operations <- controller.NewOperation(persistentVolumeClaim, controller.HighPriorityOperation, r.HeaderParameter(types.RequestIDHeader))

	w.WriteAsJson(persistentVolumeClaim)
}
