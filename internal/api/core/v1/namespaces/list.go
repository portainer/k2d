package namespaces

import (
	"github.com/emicklei/go-restful/v3"
	"github.com/portainer/k2d/internal/api/utils"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func (svc NamespaceService) ListNamespaces(r *restful.Request, w *restful.Response) {
	namespaceList := svc.adapter.ListNamespaces()

	utils.WriteListBasedOnAcceptHeader(r, w, &namespaceList, func() runtime.Object {
		return &corev1.NamespaceList{
			TypeMeta: metav1.TypeMeta{
				Kind:       "NamespaceList",
				APIVersion: "v1",
			},
		}
	}, svc.adapter.ConvertObjectToVersionedObject)
}
