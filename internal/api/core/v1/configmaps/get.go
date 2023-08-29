package configmaps

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/emicklei/go-restful/v3"
	"github.com/portainer/k2d/internal/adapter/store/filesystem"
	"github.com/portainer/k2d/internal/api/utils"
)

func (svc ConfigMapService) GetConfigMap(r *restful.Request, w *restful.Response) {
	configMapName := r.PathParameter("name")

	configMap, err := svc.adapter.GetConfigMap(configMapName)
	if err != nil && errors.Is(err, filesystem.ErrConfigMapNotFound) {
		w.WriteHeader(http.StatusNotFound)
		return
	} else if err != nil {
		utils.HttpError(r, w, http.StatusInternalServerError, fmt.Errorf("unable to get configMap: %w", err))
		return
	}

	w.WriteAsJson(configMap)
}
