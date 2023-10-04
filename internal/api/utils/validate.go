package utils

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/emicklei/go-restful/v3"
	"github.com/portainer/k2d/internal/adapter"
	adaptererr "github.com/portainer/k2d/internal/adapter/errors"
)

func ValidateNamespace(r *restful.Request, w *restful.Response, adapter *adapter.KubeDockerAdapter, namespace string) {
	// namespace validation. if doesn't exist, return 404
	_, err := adapter.GetNamespace(r.Request.Context(), namespace)
	if err != nil {
		if errors.Is(err, adaptererr.ErrResourceNotFound) {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		HttpError(r, w, http.StatusInternalServerError, fmt.Errorf("unable to get namespace: %w", err))
		return
	}
}
