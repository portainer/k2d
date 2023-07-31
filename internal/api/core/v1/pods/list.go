package pods

import (
	"fmt"
	"net/http"

	"github.com/emicklei/go-restful/v3"
	"github.com/portainer/k2d/internal/api/utils"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func (svc PodService) ListPods(r *restful.Request, w *restful.Response) {
	podList, err := svc.adapter.ListPods(r.Request.Context())
	if err != nil {
		utils.HttpError(r, w, http.StatusInternalServerError, fmt.Errorf("unable to list pods: %w", err))
		return
	}

	utils.WriteListBasedOnAcceptHeader(r, w, &podList, func() runtime.Object {
		return &corev1.PodList{
			TypeMeta: metav1.TypeMeta{
				Kind:       "PodList",
				APIVersion: "v1",
			},
		}
	}, svc.adapter.ConvertObjectToVersionedObject)
}
