package events

import (
	"github.com/emicklei/go-restful/v3"
	"github.com/portainer/k2d/internal/api/utils"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func (svc EventService) ListEvents(r *restful.Request, w *restful.Response) {
	eventList := svc.adapter.ListEvents()

	utils.WriteListBasedOnAcceptHeader(r, w, &eventList, func() runtime.Object {
		return &corev1.EventList{
			TypeMeta: metav1.TypeMeta{
				Kind:       "EventList",
				APIVersion: "v1",
			},
		}
	}, svc.adapter.ConvertObjectToVersionedObject)
}
