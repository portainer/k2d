package namespaces

import (
	"fmt"
	"net/http"

	"github.com/emicklei/go-restful/v3"
	"github.com/portainer/k2d/internal/api/utils"
	httputils "github.com/portainer/k2d/pkg/http"
	corev1 "k8s.io/api/core/v1"
)

func (svc NamespaceService) CreateNamespace(r *restful.Request, w *restful.Response) {
	namespace := &corev1.Namespace{}

	err := httputils.ParseJSONBody(r.Request, &namespace)
	if err != nil {
		utils.HttpError(r, w, http.StatusBadRequest, fmt.Errorf("unable to parse request body: %w", err))
		return
	}

	w.WriteAsJson(namespace)
}
