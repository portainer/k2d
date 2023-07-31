package deployments

import (
	"fmt"
	"net/http"

	"github.com/emicklei/go-restful/v3"
	"github.com/portainer/k2d/internal/api/utils"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func (svc DeploymentService) ListDeployments(r *restful.Request, w *restful.Response) {
	deploymentList, err := svc.adapter.ListDeployments(r.Request.Context())
	if err != nil {
		utils.HttpError(r, w, http.StatusInternalServerError, fmt.Errorf("unable to list deployments: %w", err))
		return
	}

	utils.WriteListBasedOnAcceptHeader(r, w, &deploymentList, func() runtime.Object {
		return &appsv1.DeploymentList{
			TypeMeta: metav1.TypeMeta{
				Kind:       "DeploymentList",
				APIVersion: "apps/v1",
			},
		}
	}, svc.adapter.ConvertObjectToVersionedObject)
}
