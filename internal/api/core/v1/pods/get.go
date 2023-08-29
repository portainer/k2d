package pods

import (
	"fmt"
	"net/http"

	"github.com/emicklei/go-restful/v3"
	"github.com/portainer/k2d/internal/api/utils"
)

func (svc PodService) GetPod(r *restful.Request, w *restful.Response) {
	podName := r.PathParameter("name")
	namespaceName := r.PathParameter("namespace")

	pod, err := svc.adapter.GetPod(r.Request.Context(), podName, namespaceName)
	if err != nil {
		utils.HttpError(r, w, http.StatusInternalServerError, fmt.Errorf("unable to get pod: %w", err))
		return
	}

	if pod == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	w.WriteAsJson(pod)
}
