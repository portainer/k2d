package services

import (
	"fmt"
	"net/http"

	"github.com/emicklei/go-restful/v3"
	"github.com/portainer/k2d/internal/api/utils"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (svc ServiceService) DeleteService(r *restful.Request, w *restful.Response) {
	// TODO: namespace is not implemented, there might be an issue when removing a service from another namespace
	// E.g to check
	// k create svc svc1 -n test1
	// k create svc svc1 -n test2
	// k delete svc svc1 - what happens?
	serviceName := r.PathParameter("name")

	err := svc.adapter.DeleteService(r.Request.Context(), serviceName)
	if err != nil {
		utils.HttpError(r, w, http.StatusInternalServerError, fmt.Errorf("unable to delete service: %w", err))
		return
	}

	w.WriteAsJson(metav1.Status{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Status",
			APIVersion: "v1",
		},
		Status: "Success",
		Code:   http.StatusOK,
	})
}
