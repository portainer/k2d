package pods

import (
	"net/http"

	"github.com/emicklei/go-restful/v3"
	"github.com/portainer/k2d/internal/api/utils"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (svc PodService) DeletePod(r *restful.Request, w *restful.Response) {
	namespace := utils.GetNamespaceFromRequest(r)

	podName := r.PathParameter("name")
	svc.adapter.DeletePod(r.Request.Context(), podName, namespace)

	w.WriteAsJson(metav1.Status{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Status",
			APIVersion: "v1",
		},
		Status: "Success",
		Code:   http.StatusOK,
	})
}
