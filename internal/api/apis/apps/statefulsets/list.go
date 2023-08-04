package statefulsets

import (
	"context"

	"github.com/emicklei/go-restful/v3"
	"github.com/portainer/k2d/internal/api/utils"
)

func (svc StatefulSetService) ListStatefulSets(r *restful.Request, w *restful.Response) {
	utils.ListResources(
		r,
		w,
		func(ctx context.Context) (interface{}, error) {
			return svc.adapter.ListStatefulSets(ctx)
		},
		svc.adapter.GetStatefulSetTable,
	)
}
