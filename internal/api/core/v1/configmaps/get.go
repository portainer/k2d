package configmaps

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/emicklei/go-restful/v3"
	adaptererr "github.com/portainer/k2d/internal/adapter/errors"
	"github.com/portainer/k2d/internal/api/utils"
)

func (svc ConfigMapService) GetConfigMap(r *restful.Request, w *restful.Response) {
	namespace := r.PathParameter("namespace")
	configMapName := r.PathParameter("name")

	configMap, err := svc.adapter.GetConfigMap(configMapName, namespace)
	if err != nil {
		if errors.Is(err, adaptererr.ErrResourceNotFound) {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		utils.HttpError(r, w, http.StatusInternalServerError, fmt.Errorf("unable to get configMap: %w", err))
		return
	}

	w.WriteAsJson(configMap)
}
