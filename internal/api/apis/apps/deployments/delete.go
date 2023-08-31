package deployments

import (
	"net/http"

	"github.com/emicklei/go-restful/v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (svc DeploymentService) DeleteDeployment(r *restful.Request, w *restful.Response) {
	namespace := r.PathParameter("namespace")
	deploymentName := r.PathParameter("name")

	svc.adapter.DeleteContainer(r.Request.Context(), deploymentName, namespace)

	w.WriteAsJson(metav1.Status{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Status",
			APIVersion: "v1",
		},
		Status: "Success",
		Code:   http.StatusOK,
	})
}
