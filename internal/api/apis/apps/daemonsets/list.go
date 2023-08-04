package daemonsets

import (
	"context"

	"github.com/emicklei/go-restful/v3"
	"github.com/portainer/k2d/internal/api/utils"
)

func (svc DaemonSetService) ListDaemonSets(r *restful.Request, w *restful.Response) {
	utils.ListResources(
		r,
		w,
		func(ctx context.Context) (interface{}, error) {
			return svc.adapter.ListDaemonSets(ctx)
		},
		svc.adapter.GetDaemonSetTable,
	)
}
