package converter

import (
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubernetes/pkg/apis/apps"
	"k8s.io/kubernetes/pkg/apis/core"
)

func (converter *DockerAPIConverter) ConvertContainersToStatefulSets(containers []types.Container) apps.StatefulSetList {
	statefulSets := []apps.StatefulSet{}

	for _, container := range containers {
		containerName := strings.TrimPrefix(container.Names[0], "/")
		containerState := container.State

		statefulSet := apps.StatefulSet{
			TypeMeta: metav1.TypeMeta{
				Kind:       "StatefulSet",
				APIVersion: "apps/v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:              containerName,
				CreationTimestamp: metav1.NewTime(time.Unix(container.Created, 0)),
				Namespace:         "default",
			},
			Spec: apps.StatefulSetSpec{
				Replicas: 1,
				Template: core.PodTemplateSpec{
					Spec: core.PodSpec{
						Containers: []core.Container{
							{
								Name:  containerName,
								Image: container.Image,
							},
						},
					},
				},
			},
			Status: apps.StatefulSetStatus{
				Replicas: 1,
			},
		}

		if containerState == "running" {
			statefulSet.Status.ReadyReplicas = 1
			statefulSet.Status.CurrentReplicas = 1
			statefulSet.Status.UpdatedReplicas = 1
			statefulSet.Status.AvailableReplicas = 1
		}

		statefulSets = append(statefulSets, statefulSet)
	}

	return apps.StatefulSetList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "StatefulSetList",
			APIVersion: "apps/v1",
		},
		Items: statefulSets,
	}
}
