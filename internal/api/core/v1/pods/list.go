package pods

import (
	"context"

	"github.com/emicklei/go-restful/v3"
	"github.com/portainer/k2d/internal/api/utils"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (svc PodService) ListPods(r *restful.Request, w *restful.Response) {
	namespaceName := r.PathParameter("namespace")
	utils.ListResources(
		r,
		w,
		func(ctx context.Context) (interface{}, error) {
			return svc.adapter.ListPods(ctx, namespaceName)
		},
		func(ctx context.Context) (*metav1.Table, error) {
			return svc.adapter.GetPodTable(ctx, namespaceName)
		},
	)
}
