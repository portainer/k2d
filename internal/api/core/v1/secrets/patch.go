package secrets

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

func (svc SecretService) PatchSecret(r *restful.Request, w *restful.Response) {
	namespace := r.PathParameter("namespace")
	// namespace validation. if doesn't exist, return 404
	utils.ValidateNamespace(r, w, svc.adapter, namespace)

	secretName := r.PathParameter("name")
	patch, err := io.ReadAll(r.Request.Body)
	if err != nil {
		utils.HttpError(r, w, http.StatusBadRequest, fmt.Errorf("unable to parse request body: %w", err))
		return
	}

	secret, err := svc.adapter.GetSecret(secretName, namespace)
	if err != nil && errors.Is(err, adaptererr.ErrResourceNotFound) {
		w.WriteHeader(http.StatusNotFound)
		return
	} else if err != nil {
		utils.HttpError(r, w, http.StatusInternalServerError, fmt.Errorf("unable to get secret: %w", err))
		return
	}

	data, err := json.Marshal(secret)
	if err != nil {
		utils.HttpError(r, w, http.StatusInternalServerError, fmt.Errorf("unable to marshal secret: %w", err))
		return
	}

	mergedData, err := strategicpatch.StrategicMergePatch(data, patch, corev1.Secret{})
	if err != nil {
		utils.HttpError(r, w, http.StatusInternalServerError, fmt.Errorf("unable to apply patch: %w", err))
		return
	}

	updatedSecret := &corev1.Secret{}

	err = json.Unmarshal(mergedData, updatedSecret)
	if err != nil {
		utils.HttpError(r, w, http.StatusInternalServerError, fmt.Errorf("unable to unmarshal secret: %w", err))
		return
	}

	dryRun := r.QueryParameter("dryRun") != ""
	if dryRun {
		w.WriteAsJson(updatedSecret)
		return
	}

	svc.operations <- controller.NewOperation(updatedSecret, controller.HighPriorityOperation, r.HeaderParameter(types.RequestIDHeader))

	w.WriteAsJson(updatedSecret)
}
