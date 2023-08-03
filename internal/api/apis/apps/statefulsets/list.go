package statefulsets

import (
	"fmt"
	"net/http"

	"github.com/emicklei/go-restful/v3"
	"github.com/portainer/k2d/internal/api/utils"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func (svc StatefulSetService) ListStatefulSets(r *restful.Request, w *restful.Response) {
	statefulSetList, err := svc.adapter.ListStatefulSets(r.Request.Context())
	if err != nil {
		utils.HttpError(r, w, http.StatusInternalServerError, fmt.Errorf("unable to list statefulsets: %w", err))
		return
	}

	utils.WriteListBasedOnAcceptHeader(r, w, &statefulSetList, func() runtime.Object {
		return &appsv1.StatefulSetList{
			TypeMeta: metav1.TypeMeta{
				Kind:       "StatefulSetList",
				APIVersion: "apps/v1",
			},
		}
	}, svc.adapter.ConvertObjectToVersionedObject)
}
