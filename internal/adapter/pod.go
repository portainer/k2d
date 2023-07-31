package adapter

import (
	"context"
	"fmt"
	"io"

	"github.com/docker/docker/api/types"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubernetes/pkg/apis/core"
)

type PodLogOptions struct {
	Timestamps bool
	Follow     bool
	Tail       string
}

func (adapter *KubeDockerAdapter) CreateContainerFromPod(ctx context.Context, pod *corev1.Pod) error {
	opts := ContainerCreationOptions{
		containerName: pod.Name,
		podSpec:       pod.Spec,
		labels:        pod.Labels,
	}

	opts.lastAppliedConfiguration = pod.ObjectMeta.Annotations["kubectl.kubernetes.io/last-applied-configuration"]

	return adapter.createContainerFromPodSpec(ctx, opts)
}

func (adapter *KubeDockerAdapter) GetPod(ctx context.Context, podName string) (*corev1.Pod, error) {
	containers, err := adapter.cli.ContainerList(ctx, types.ContainerListOptions{All: true})
	if err != nil {
		return nil, fmt.Errorf("unable to list containers: %w", err)
	}

	for _, container := range containers {
		if container.Names[0] == "/"+podName {
			pod := adapter.converter.ConvertContainerToPod(container)

			versionedPod := corev1.Pod{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Pod",
					APIVersion: "v1",
				},
			}

			err := adapter.ConvertObjectToVersionedObject(&pod, &versionedPod)
			if err != nil {
				return nil, fmt.Errorf("unable to convert object to versioned object: %w", err)
			}

			return &versionedPod, nil
		}
	}

	return nil, nil
}

func (adapter *KubeDockerAdapter) GetPodLogs(ctx context.Context, podName string, opts PodLogOptions) (io.ReadCloser, error) {
	containers, err := adapter.cli.ContainerList(ctx, types.ContainerListOptions{All: true})
	if err != nil {
		return nil, fmt.Errorf("unable to list containers: %w", err)
	}

	for _, container := range containers {
		if container.Names[0] == "/"+podName {
			return adapter.cli.ContainerLogs(ctx, container.ID, types.ContainerLogsOptions{
				ShowStdout: true,
				ShowStderr: true,
				Timestamps: opts.Timestamps,
				Follow:     opts.Follow,
				Tail:       opts.Tail,
			})
		}
	}

	return nil, nil
}

func (adapter *KubeDockerAdapter) ListPods(ctx context.Context) (core.PodList, error) {
	containers, err := adapter.cli.ContainerList(ctx, types.ContainerListOptions{All: true})
	if err != nil {
		return core.PodList{}, fmt.Errorf("unable to list containers: %w", err)
	}

	podList := adapter.converter.ConvertContainersToPods(containers)

	return podList, nil
}
