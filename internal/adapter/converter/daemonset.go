package converter

import (
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubernetes/pkg/apis/apps"
	"k8s.io/kubernetes/pkg/apis/core"
)

func (converter *DockerAPIConverter) ConvertContainersToDaemonSets(containers []types.Container) apps.DaemonSetList {
	daemonSets := []apps.DaemonSet{}

	for _, container := range containers {
		containerName := strings.TrimPrefix(container.Names[0], "/")
		containerState := container.State

		daemonSet := apps.DaemonSet{
			TypeMeta: metav1.TypeMeta{
				Kind:       "DaemonSet",
				APIVersion: "apps/v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:              containerName,
				CreationTimestamp: metav1.NewTime(time.Unix(container.Created, 0)),
				Namespace:         "default",
			},
			Spec: apps.DaemonSetSpec{
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
			Status: apps.DaemonSetStatus{
				DesiredNumberScheduled: 1,
			},
		}

		if containerState == "running" {
			daemonSet.Status.UpdatedNumberScheduled = 1
			daemonSet.Status.CurrentNumberScheduled = 1
			daemonSet.Status.NumberAvailable = 1
			daemonSet.Status.NumberReady = 1
		} else {
			daemonSet.Status.NumberUnavailable = 1
		}

		daemonSets = append(daemonSets, daemonSet)
	}

	return apps.DaemonSetList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "DaemonSetList",
			APIVersion: "apps/v1",
		},
		Items: daemonSets,
	}
}
