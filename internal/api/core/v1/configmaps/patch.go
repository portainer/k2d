package configmaps

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/emicklei/go-restful/v3"
	adaptererr "github.com/portainer/k2d/internal/adapter/errors"
	"github.com/portainer/k2d/internal/api/utils"
	"github.com/portainer/k2d/internal/controller"
	"github.com/portainer/k2d/internal/types"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
)

func (svc ConfigMapService) PatchConfigMap(r *restful.Request, w *restful.Response) {
	namespace := r.PathParameter("namespace")
	configMapName := r.PathParameter("name")

	patch, err := io.ReadAll(r.Request.Body)
	if err != nil {
		utils.HttpError(r, w, http.StatusBadRequest, fmt.Errorf("unable to parse request body: %w", err))
		return
	}

	configMap, err := svc.adapter.GetConfigMap(configMapName, namespace)
	if err != nil && errors.Is(err, adaptererr.ErrResourceNotFound) {
		w.WriteHeader(http.StatusNotFound)
		return
	} else if err != nil {
		utils.HttpError(r, w, http.StatusInternalServerError, fmt.Errorf("unable to get configMap: %w", err))
		return
	}

	data, err := json.Marshal(configMap)
	if err != nil {
		utils.HttpError(r, w, http.StatusInternalServerError, fmt.Errorf("unable to marshal configMap: %w", err))
		return
	}

	mergedData, err := strategicpatch.StrategicMergePatch(data, patch, corev1.ConfigMap{})
	if err != nil {
		utils.HttpError(r, w, http.StatusInternalServerError, fmt.Errorf("unable to apply patch: %w", err))
		return
	}

	updatedConfigMap := &corev1.ConfigMap{}

	err = json.Unmarshal(mergedData, updatedConfigMap)
	if err != nil {
		utils.HttpError(r, w, http.StatusInternalServerError, fmt.Errorf("unable to unmarshal configMap: %w", err))
		return
	}

	dryRun := r.QueryParameter("dryRun") != ""
	if dryRun {
		w.WriteAsJson(updatedConfigMap)
		return
	}

	svc.operations <- controller.NewOperation(updatedConfigMap, controller.HighPriorityOperation, r.HeaderParameter(types.RequestIDHeader))

	w.WriteAsJson(updatedConfigMap)
}
