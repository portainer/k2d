package adapter

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/docker/docker/api/types"
	k2dtypes "github.com/portainer/k2d/internal/adapter/types"
	"github.com/portainer/k2d/internal/k8s"
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

	if pod.Labels["app.kubernetes.io/managed-by"] == "Helm" {
		podData, err := json.Marshal(pod)
		if err != nil {
			return fmt.Errorf("unable to marshal pod: %w", err)
		}
		pod.ObjectMeta.Annotations["kubectl.kubernetes.io/last-applied-configuration"] = string(podData)
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
			pod, err := adapter.getPod(container)
			if err != nil {
				return nil, fmt.Errorf("unable to get pod: %w", err)
			}

			versionedPod := corev1.Pod{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Pod",
					APIVersion: "v1",
				},
			}

			err = adapter.ConvertK8SResource(pod, &versionedPod)
			if err != nil {
				return nil, fmt.Errorf("unable to convert internal object to versioned object: %w", err)
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

func (adapter *KubeDockerAdapter) GetPodTable(ctx context.Context) (*metav1.Table, error) {
	podList, err := adapter.listPods(ctx)
	if err != nil {
		return &metav1.Table{}, fmt.Errorf("unable to list pods: %w", err)
	}

	return k8s.GenerateTable(&podList)
}

func (adapter *KubeDockerAdapter) ListPods(ctx context.Context) (corev1.PodList, error) {
	podList, err := adapter.listPods(ctx)
	if err != nil {
		return corev1.PodList{}, fmt.Errorf("unable to list pods: %w", err)
	}

	versionedPodList := corev1.PodList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "PodList",
			APIVersion: "v1",
		},
	}

	err = adapter.ConvertK8SResource(&podList, &versionedPodList)
	if err != nil {
		return corev1.PodList{}, fmt.Errorf("unable to convert internal PodList to versioned PodList: %w", err)
	}

	return versionedPodList, nil
}

// Retrieving a pod uses a different approach than the other resources.
// We build a Pod object from the container details by default and then we replace
// the pod spec with the one stored in the container labels if it exists.
// This is to keep the ability to list pods that were created outside of k2d (such as via docker run).
func (adapter *KubeDockerAdapter) getPod(container types.Container) (*core.Pod, error) {
	pod := adapter.converter.ConvertContainerToPod(container)

	if container.Labels[k2dtypes.PodLastAppliedConfigLabelKey] != "" {
		internalPodSpecData := container.Labels[k2dtypes.PodLastAppliedConfigLabelKey]
		podSpec := core.PodSpec{}

		err := json.Unmarshal([]byte(internalPodSpecData), &podSpec)
		if err != nil {
			return nil, fmt.Errorf("unable to unmarshal pod spec: %w", err)
		}

		pod.Spec = podSpec
	}

	return &pod, nil
}

func (adapter *KubeDockerAdapter) listPods(ctx context.Context) (core.PodList, error) {
	containers, err := adapter.cli.ContainerList(ctx, types.ContainerListOptions{All: true})
	if err != nil {
		return core.PodList{}, fmt.Errorf("unable to list containers: %w", err)
	}

	pods := []core.Pod{}

	for _, container := range containers {
		pod, err := adapter.getPod(container)
		if err != nil {
			return core.PodList{}, fmt.Errorf("unable to get pods: %w", err)
		}

		pods = append(pods, *pod)
	}

	return core.PodList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "PodList",
			APIVersion: "v1",
		},
		Items: pods,
	}, nil
}
