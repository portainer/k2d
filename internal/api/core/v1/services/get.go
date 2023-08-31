package services

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/emicklei/go-restful/v3"
	adaptererr "github.com/portainer/k2d/internal/adapter/errors"
	"github.com/portainer/k2d/internal/api/utils"
)

func (svc ServiceService) GetService(r *restful.Request, w *restful.Response) {
	namespace := r.PathParameter("namespace")
	serviceName := r.PathParameter("name")

	service, err := svc.adapter.GetService(r.Request.Context(), serviceName, namespace)
	if err != nil {
		if errors.Is(err, adaptererr.ErrResourceNotFound) {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		utils.HttpError(r, w, http.StatusInternalServerError, fmt.Errorf("unable to get service: %w", err))
		return
	}

	w.WriteAsJson(service)
}
