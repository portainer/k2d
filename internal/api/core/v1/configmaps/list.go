package configmaps

import (
	"context"

	"github.com/emicklei/go-restful/v3"
	"github.com/portainer/k2d/internal/api/utils"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (svc ConfigMapService) ListConfigMaps(r *restful.Request, w *restful.Response) {
	utils.ListResources(
		r,
		w,
		func(ctx context.Context) (interface{}, error) {
			return svc.adapter.ListConfigMaps()
		},
		func(ctx context.Context) (*metav1.Table, error) {
			return svc.adapter.GetConfigMapTable()
		},
	)
}
