package pods

import (
	"net/http"

	"github.com/emicklei/go-restful/v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (svc PodService) DeletePod(r *restful.Request, w *restful.Response) {
	namespace := r.PathParameter("namespace")
	podName := r.PathParameter("name")

	svc.adapter.DeleteContainer(r.Request.Context(), podName, namespace)

	w.WriteAsJson(metav1.Status{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Status",
			APIVersion: "v1",
		},
		Status: "Success",
		Code:   http.StatusOK,
	})
}
