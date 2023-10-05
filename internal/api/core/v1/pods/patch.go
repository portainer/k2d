package pods

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

func (svc PodService) PatchPod(r *restful.Request, w *restful.Response) {
	namespace := utils.GetNamespaceFromRequest(r)

	podName := r.PathParameter("name")
	patch, err := io.ReadAll(r.Request.Body)
	if err != nil {
		utils.HttpError(r, w, http.StatusBadRequest, fmt.Errorf("unable to parse request body: %w", err))
		return
	}

	pod, err := svc.adapter.GetPod(r.Request.Context(), podName, namespace)
	if err != nil {
		utils.HttpError(r, w, http.StatusInternalServerError, fmt.Errorf("unable to get pod: %w", err))
		return
	}

	if pod == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	data, err := json.Marshal(pod)
	if err != nil {
		utils.HttpError(r, w, http.StatusInternalServerError, fmt.Errorf("unable to marshal pod: %w", err))
		return
	}

	mergedData, err := strategicpatch.StrategicMergePatch(data, patch, corev1.Pod{})
	if err != nil {
		utils.HttpError(r, w, http.StatusInternalServerError, fmt.Errorf("unable to apply patch: %w", err))
		return
	}

	updatedPod := &corev1.Pod{}

	err = json.Unmarshal(mergedData, updatedPod)
	if err != nil {
		utils.HttpError(r, w, http.StatusInternalServerError, fmt.Errorf("unable to unmarshal pod: %w", err))
		return
	}

	dryRun := r.QueryParameter("dryRun") != ""
	if dryRun {
		w.WriteAsJson(updatedPod)
		return
	}

	svc.operations <- controller.NewOperation(updatedPod, controller.MediumPriorityOperation, r.HeaderParameter(types.RequestIDHeader))

	w.WriteAsJson(updatedPod)
}
