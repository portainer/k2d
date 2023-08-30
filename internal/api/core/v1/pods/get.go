package pods

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/emicklei/go-restful/v3"
	adaptererr "github.com/portainer/k2d/internal/adapter/errors"
	"github.com/portainer/k2d/internal/api/utils"
)

func (svc PodService) GetPod(r *restful.Request, w *restful.Response) {
	namespace := utils.NamespaceParameter(r)
	podName := r.PathParameter("name")

	pod, err := svc.adapter.GetPod(r.Request.Context(), podName, namespace)
	if err != nil {
		if errors.Is(err, adaptererr.ErrResourceNotFound) {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		utils.HttpError(r, w, http.StatusInternalServerError, fmt.Errorf("unable to get pod: %w", err))
		return
	}

	w.WriteAsJson(pod)
}
