package services

import (
	"fmt"
	"net/http"

	"github.com/emicklei/go-restful/v3"
	"github.com/portainer/k2d/internal/api/utils"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func (svc ServiceService) ListServices(r *restful.Request, w *restful.Response) {
	serviceList, err := svc.adapter.ListServices(r.Request.Context())
	if err != nil {
		utils.HttpError(r, w, http.StatusInternalServerError, fmt.Errorf("unable to list services: %w", err))
		return
	}

	utils.WriteListBasedOnAcceptHeader(r, w, &serviceList, func() runtime.Object {
		return &corev1.ServiceList{
			TypeMeta: metav1.TypeMeta{
				Kind:       "ServiceList",
				APIVersion: "v1",
			},
		}
	}, svc.adapter.ConvertObjectToVersionedObject)
}
