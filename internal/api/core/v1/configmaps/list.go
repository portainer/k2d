package configmaps

import (
	"fmt"
	"net/http"

	"github.com/emicklei/go-restful/v3"
	"github.com/portainer/k2d/internal/api/utils"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func (svc ConfigMapService) ListConfigMaps(r *restful.Request, w *restful.Response) {
	configMapList, err := svc.adapter.ListConfigMaps()
	if err != nil {
		utils.HttpError(r, w, http.StatusInternalServerError, fmt.Errorf("unable to list configMaps: %w", err))
		return
	}

	utils.WriteListBasedOnAcceptHeader(r, w, &configMapList, func() runtime.Object {
		return &corev1.ConfigMapList{
			TypeMeta: metav1.TypeMeta{
				Kind:       "ConfigMapList",
				APIVersion: "v1",
			},
		}
	}, svc.adapter.ConvertObjectToVersionedObject)
}
