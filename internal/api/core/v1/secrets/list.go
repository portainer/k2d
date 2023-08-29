package secrets

import (
	"context"
	"fmt"
	"net/http"

	"github.com/emicklei/go-restful/v3"
	"github.com/portainer/k2d/internal/api/utils"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

func (svc SecretService) ListSecrets(r *restful.Request, w *restful.Response) {
	namespace := utils.NamespaceParameter(r)
	selectorParam := r.QueryParameter("labelSelector")

	selector, err := labels.Parse(selectorParam)
	if err != nil {
		utils.HttpError(r, w, http.StatusBadRequest, fmt.Errorf("invalid selector parameter: %w", err))
		return
	}

	utils.ListResources(
		r,
		w,
		func(ctx context.Context) (interface{}, error) {
			return svc.adapter.ListSecrets(namespace, selector)
		},
		func(ctx context.Context) (*metav1.Table, error) {
			return svc.adapter.GetSecretTable(namespace, selector)
		},
	)
}
