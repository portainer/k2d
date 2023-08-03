package daemonsets

import (
	"fmt"
	"net/http"

	"github.com/emicklei/go-restful/v3"
	"github.com/portainer/k2d/internal/api/utils"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func (svc DaemonSetService) ListDaemonSets(r *restful.Request, w *restful.Response) {
	daemonSetList, err := svc.adapter.ListDaemonSets(r.Request.Context())
	if err != nil {
		utils.HttpError(r, w, http.StatusInternalServerError, fmt.Errorf("unable to list daemonsets: %w", err))
		return
	}

	utils.WriteListBasedOnAcceptHeader(r, w, &daemonSetList, func() runtime.Object {
		return &appsv1.DaemonSetList{
			TypeMeta: metav1.TypeMeta{
				Kind:       "DaemonSetList",
				APIVersion: "apps/v1",
			},
		}
	}, svc.adapter.ConvertObjectToVersionedObject)
}
