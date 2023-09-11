package persistentvolumeclaims

import (
	"fmt"
	"net/http"

	"github.com/emicklei/go-restful/v3"
	"github.com/portainer/k2d/internal/api/utils"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (svc PersistentVolumeClaimService) DeletePersistentVolumeClaim(r *restful.Request, w *restful.Response) {
	namespace := r.PathParameter("namespace")
	persistentVolumeClaimName := r.PathParameter("name")

	err := svc.adapter.DeletePersistentVolumeClaim(r.Request.Context(), persistentVolumeClaimName, namespace)
	if err != nil {
		utils.HttpError(r, w, http.StatusInternalServerError, fmt.Errorf("unable to delete persistent volume claim: %w", err))
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
