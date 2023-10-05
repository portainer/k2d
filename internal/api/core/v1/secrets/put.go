package secrets

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/emicklei/go-restful/v3"
	adaptererr "github.com/portainer/k2d/internal/adapter/errors"
	"github.com/portainer/k2d/internal/api/utils"
	httputils "github.com/portainer/k2d/pkg/http"
	corev1 "k8s.io/api/core/v1"
)

func (svc SecretService) PutSecret(r *restful.Request, w *restful.Response) {
	namespace := utils.GetNamespaceFromRequest(r)

	secretName := r.PathParameter("name")
	secret := &corev1.Secret{}
	err := httputils.ParseJSONBody(r.Request, &secret)
	if err != nil {
		utils.HttpError(r, w, http.StatusBadRequest, fmt.Errorf("unable to parse request body: %w", err))
		return
	}

	dryRun := r.QueryParameter("dryRun") != ""
	if dryRun {
		w.WriteAsJson(secret)
		return
	}

	// TODO: this is a temporary hack to to wait for secrets to be created
	// by clients that are sending rapid requests such as Helm
	// To work around this, we introduce a retry mechanism that will
	// look for the secret every second and retry for 10 seconds
	timeoutCh := time.After(10 * time.Second)

	for {
		_, err := svc.adapter.GetSecret(secretName, namespace)
		if err == nil {
			// The secret has been found, we can update it
			err = svc.adapter.CreateSecret(secret)
			if err != nil {
				utils.HttpError(r, w, http.StatusInternalServerError, fmt.Errorf("unable to update secret: %w", err))
				return
			}

			w.WriteAsJson(secret)
			return
		}

		if err != nil && !errors.Is(err, adaptererr.ErrResourceNotFound) {
			utils.HttpError(r, w, http.StatusInternalServerError, fmt.Errorf("unable to get secret: %w", err))
			return
		}

		// Wait for either the delay for retry or a timeout
		select {
		case <-time.After(1 * time.Second):
			continue
		case <-timeoutCh:
			w.WriteHeader(http.StatusNotFound)
			return
		}
	}
}
