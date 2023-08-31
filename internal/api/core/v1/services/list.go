package services

import (
	"context"

	"github.com/emicklei/go-restful/v3"
	"github.com/portainer/k2d/internal/api/utils"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (svc ServiceService) ListServices(r *restful.Request, w *restful.Response) {
	namespace := utils.NamespaceParameter(r)

	utils.ListResources(
		r,
		w,
		func(ctx context.Context) (interface{}, error) {
			return svc.adapter.ListServices(ctx, namespace)
		},
		func(ctx context.Context) (*metav1.Table, error) {
			return svc.adapter.GetServiceTable(ctx, namespace)
		},
	)
}
