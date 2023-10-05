package secrets

import (
	"fmt"
	"net/http"

	"github.com/emicklei/go-restful/v3"
	"github.com/portainer/k2d/internal/api/utils"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (svc SecretService) DeleteSecret(r *restful.Request, w *restful.Response) {
	namespace := utils.GetNamespaceFromRequest(r)

	secretName := r.PathParameter("name")
	err := svc.adapter.DeleteSecret(secretName, namespace)
	if err != nil {
		utils.HttpError(r, w, http.StatusInternalServerError, fmt.Errorf("unable to delete secret: %w", err))
		return
	}

	w.WriteAsJson(metav1.Status{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Status",
			APIVersion: "v1",
		},
		Status: "Success",
		Code:   http.StatusOK,
	})
}
