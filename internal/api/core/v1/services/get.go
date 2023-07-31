package services

import (
	"fmt"
	"net/http"

	"github.com/emicklei/go-restful/v3"
	"github.com/portainer/k2d/internal/api/utils"
)

func (svc ServiceService) GetService(r *restful.Request, w *restful.Response) {
	serviceName := r.PathParameter("name")

	service, err := svc.adapter.GetService(r.Request.Context(), serviceName)
	if err != nil {
		utils.HttpError(r, w, http.StatusInternalServerError, fmt.Errorf("unable to get service: %w", err))
		return
	}

	if service == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	w.WriteAsJson(service)
}
