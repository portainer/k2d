package persistentvolumeclaims

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

func (svc PersistentVolumeClaimService) PatchPersistentVolumeClaim(r *restful.Request, w *restful.Response) {
	namespace := utils.NamespaceParameter(r)
	persistentVolumeClaimName := r.PathParameter("name")

	patch, err := io.ReadAll(r.Request.Body)
	if err != nil {
		utils.HttpError(r, w, http.StatusBadRequest, fmt.Errorf("unable to parse request body: %w", err))
		return
	}

	persistentVolumeClaim, err := svc.adapter.GetPersistentVolumeClaim(r.Request.Context(), persistentVolumeClaimName, namespace)
	if err != nil {
		utils.HttpError(r, w, http.StatusInternalServerError, fmt.Errorf("unable to get persistent volume claim: %w", err))
		return
	}

	data, err := json.Marshal(persistentVolumeClaim)
	if err != nil {
		utils.HttpError(r, w, http.StatusInternalServerError, fmt.Errorf("unable to marshal persistent volume claim: %w", err))
		return
	}

	mergedData, err := strategicpatch.StrategicMergePatch(data, patch, corev1.PersistentVolumeClaim{})
	if err != nil {
		utils.HttpError(r, w, http.StatusInternalServerError, fmt.Errorf("unable to apply patch: %w", err))
		return
	}

	updatedPersistentVolumeClaim := &corev1.PersistentVolumeClaim{}

	err = json.Unmarshal(mergedData, updatedPersistentVolumeClaim)
	if err != nil {
		utils.HttpError(r, w, http.StatusInternalServerError, fmt.Errorf("unable to unmarshal namespace: %w", err))
		return
	}

	dryRun := r.QueryParameter("dryRun") != ""
	if dryRun {
		w.WriteAsJson(updatedPersistentVolumeClaim)
		return
	}

	svc.operations <- controller.NewOperation(updatedPersistentVolumeClaim, controller.HighPriorityOperation, r.HeaderParameter(types.RequestIDHeader))

	w.WriteAsJson(updatedPersistentVolumeClaim)
}
