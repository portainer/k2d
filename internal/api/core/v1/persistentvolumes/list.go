package persistentvolumes

import (
	"context"

	"github.com/emicklei/go-restful/v3"
	"github.com/portainer/k2d/internal/api/utils"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (svc PersistentVolumeService) ListPersistentVolumes(r *restful.Request, w *restful.Response) {
	utils.ListResources(
		r,
		w,
		func(ctx context.Context) (interface{}, error) {
			return svc.adapter.ListPersistentVolumes(ctx)
		},
		func(ctx context.Context) (*metav1.Table, error) {
			return svc.adapter.GetPersistentVolumeTable(ctx)
		},
	)
}
