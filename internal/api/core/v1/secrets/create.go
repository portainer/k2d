package secrets

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

func (svc SecretService) CreateSecret(r *restful.Request, w *restful.Response) {
	namespace := utils.GetNamespaceFromRequest(r)

	secret := &corev1.Secret{}
	err := httputils.ParseJSONBody(r.Request, &secret)
	if err != nil {
		utils.HttpError(r, w, http.StatusBadRequest, fmt.Errorf("unable to parse request body: %w", err))
		return
	}

	secret.Namespace = namespace

	dryRun := r.QueryParameter("dryRun") != ""
	if dryRun {
		w.WriteAsJson(secret)
		return
	}

	svc.operations <- controller.NewOperation(secret, controller.HighPriorityOperation, r.HeaderParameter(types.RequestIDHeader))

	w.WriteAsJson(secret)
}
