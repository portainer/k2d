package configmaps

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/emicklei/go-restful/v3"
	storeerr "github.com/portainer/k2d/internal/adapter/store/errors"
	"github.com/portainer/k2d/internal/api/utils"
)

func (svc ConfigMapService) GetConfigMap(r *restful.Request, w *restful.Response) {
	namespace := utils.NamespaceParameter(r)
	configMapName := r.PathParameter("name")

	// TODO: refactor all this to use the if err != nil pattern and then if errors.Is(err, storeerr.ErrResourceNotFound) { ... }
	configMap, err := svc.adapter.GetConfigMap(configMapName, namespace)
	if err != nil && errors.Is(err, storeerr.ErrResourceNotFound) {
		w.WriteHeader(http.StatusNotFound)
		return
	} else if err != nil {
		utils.HttpError(r, w, http.StatusInternalServerError, fmt.Errorf("unable to get configMap: %w", err))
		return
	}

	w.WriteAsJson(configMap)
}
