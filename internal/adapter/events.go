package adapter

import (
	"fmt"

	"github.com/portainer/k2d/internal/k8s"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubernetes/pkg/apis/core"
)

func (adapter *KubeDockerAdapter) ListEvents() (corev1.EventList, error) {
	eventList := adapter.listEvents()

	versionedEventList := corev1.EventList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "EventList",
			APIVersion: "v1",
		},
	}

	err := adapter.ConvertK8SResource(&eventList, &versionedEventList)
	if err != nil {
		return corev1.EventList{}, fmt.Errorf("unable to convert internal EventList to versioned EventList: %w", err)
	}

	return versionedEventList, nil
}

func (adapter *KubeDockerAdapter) GetEventTable() (*metav1.Table, error) {
	eventList := adapter.listEvents()
	return k8s.GenerateTable(&eventList)
}

func (adapter *KubeDockerAdapter) listEvents() core.EventList {
	return core.EventList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "EventList",
			APIVersion: "v1",
		},
		Items: []core.Event{},
	}
}
