package storageclasses

import (
	"context"

	"github.com/emicklei/go-restful/v3"
	"github.com/portainer/k2d/internal/api/utils"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (svc StorageClassService) ListStorageClass(r *restful.Request, w *restful.Response) {
	utils.ListResources(
		r,
		w,
		func(ctx context.Context) (interface{}, error) {
			return svc.adapter.ListStorageClasses(r.Request.Context())
		},
		func(ctx context.Context) (*metav1.Table, error) {
			return svc.adapter.GetStorageClassTable(r.Request.Context())
		},
	)
}
