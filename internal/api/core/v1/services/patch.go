package services

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/emicklei/go-restful/v3"
	"github.com/portainer/k2d/internal/api/utils"
	"github.com/portainer/k2d/internal/controller"
	"github.com/portainer/k2d/internal/types"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
)

func (svc ServiceService) PatchService(r *restful.Request, w *restful.Response) {
	namespace := r.PathParameter("namespace")
	serviceName := r.PathParameter("name")

	patch, err := io.ReadAll(r.Request.Body)
	if err != nil {
		utils.HttpError(r, w, http.StatusBadRequest, fmt.Errorf("unable to parse request body: %w", err))
		return
	}

	service, err := svc.adapter.GetService(r.Request.Context(), serviceName, namespace)
	if err != nil {
		utils.HttpError(r, w, http.StatusInternalServerError, fmt.Errorf("unable to get service: %w", err))
		return
	}

	if service == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	data, err := json.Marshal(service)
	if err != nil {
		utils.HttpError(r, w, http.StatusInternalServerError, fmt.Errorf("unable to marshal service: %w", err))
		return
	}

	mergedData, err := strategicpatch.StrategicMergePatch(data, patch, corev1.Service{})
	if err != nil {
		utils.HttpError(r, w, http.StatusInternalServerError, fmt.Errorf("unable to apply patch: %w", err))
		return
	}

	updatedService := &corev1.Service{}

	err = json.Unmarshal(mergedData, updatedService)
	if err != nil {
		utils.HttpError(r, w, http.StatusInternalServerError, fmt.Errorf("unable to unmarshal service: %w", err))
		return
	}

	dryRun := r.QueryParameter("dryRun") != ""
	if dryRun {
		w.WriteAsJson(updatedService)
		return
	}

	svc.operations <- controller.NewOperation(updatedService, controller.LowPriorityOperation, r.HeaderParameter(types.RequestIDHeader))

	w.WriteAsJson(updatedService)
}
