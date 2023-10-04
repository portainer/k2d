package namespaces

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

func (svc NamespaceService) PatchNamespace(r *restful.Request, w *restful.Response) {
	namespaceName := r.PathParameter("namespace")
	// namespace validation. if doesn't exist, return 404
	utils.ValidateNamespace(r, w, svc.adapter, namespaceName)

	patch, err := io.ReadAll(r.Request.Body)
	if err != nil {
		utils.HttpError(r, w, http.StatusBadRequest, fmt.Errorf("unable to parse request body: %w", err))
		return
	}

	namespace, err := svc.adapter.GetNamespace(r.Request.Context(), namespaceName)
	if err != nil {
		utils.HttpError(r, w, http.StatusInternalServerError, fmt.Errorf("unable to get namespace: %w", err))
		return
	}

	data, err := json.Marshal(namespace)
	if err != nil {
		utils.HttpError(r, w, http.StatusInternalServerError, fmt.Errorf("unable to marshal namespace: %w", err))
		return
	}

	mergedData, err := strategicpatch.StrategicMergePatch(data, patch, corev1.Namespace{})
	if err != nil {
		utils.HttpError(r, w, http.StatusInternalServerError, fmt.Errorf("unable to apply patch: %w", err))
		return
	}

	updatedNamespace := &corev1.Namespace{}

	err = json.Unmarshal(mergedData, updatedNamespace)
	if err != nil {
		utils.HttpError(r, w, http.StatusInternalServerError, fmt.Errorf("unable to unmarshal namespace: %w", err))
		return
	}

	dryRun := r.QueryParameter("dryRun") != ""
	if dryRun {
		w.WriteAsJson(updatedNamespace)
		return
	}

	svc.operations <- controller.NewOperation(updatedNamespace, controller.HighPriorityOperation, r.HeaderParameter(types.RequestIDHeader))

	w.WriteAsJson(updatedNamespace)
}
