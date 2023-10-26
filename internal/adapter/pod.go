package adapter

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/docker/docker/api/types"
	"github.com/portainer/k2d/internal/k8s"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type PodLogOptions struct {
	Timestamps bool
	Follow     bool
	Tail       string
}

func (adapter *KubeDockerAdapter) CreateContainerFromPod(ctx context.Context, pod *corev1.Pod) error {
	opts := ContainerCreationOptions{
		containerName: pod.Name,
		namespace:     pod.Namespace,
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

func (adapter *KubeDockerAdapter) DeletePod(ctx context.Context, podName string, namespace string) error {
	container, err := adapter.findContainerFromPodAndNamespace(ctx, podName, namespace)
	if err != nil {
		return fmt.Errorf("unable to find container associated to the pod %s/%s: %w", namespace, podName, err)
	}

	err = adapter.cli.ContainerRemove(ctx, container.Names[0], types.ContainerRemoveOptions{Force: true})
	if err != nil {
		adapter.logger.Warnf("unable to remove container: %s", err)
	}

	return nil
}

// The GetPod implementation is using a filtered list approach as the Docker API provide different response types
// when inspecting a container and listing containers.
// The logic used to build a pod from a container is based on the type returned by the list operation (types.Container)
// and not the inspect operation (types.ContainerJSON).
// This is because using the inspect operation everywhere would be more expensive overall.
func (adapter *KubeDockerAdapter) GetPod(ctx context.Context, podName string, namespace string) (*corev1.Pod, error) {
	container, err := adapter.findContainerFromPodAndNamespace(ctx, podName, namespace)
	if err != nil {
		return nil, fmt.Errorf("unable to find container associated to the pod %s/%s: %w", namespace, podName, err)
	}

	pod, err := adapter.buildPodFromContainer(*container)
	if err != nil {
		return nil, fmt.Errorf("unable to get pod: %w", err)
	}

	versionedPod := corev1.Pod{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Pod",
			APIVersion: "v1",
		},
	}

	err = adapter.ConvertK8SResource(&pod, &versionedPod)
	if err != nil {
		return nil, fmt.Errorf("unable to convert internal object to versioned object: %w", err)
	}

	return &versionedPod, nil
}

func (adapter *KubeDockerAdapter) GetPodLogs(ctx context.Context, namespace string, podName string, opts PodLogOptions) (io.ReadCloser, error) {
	container, err := adapter.findContainerFromPodAndNamespace(ctx, podName, namespace)
	if err != nil {
		return nil, fmt.Errorf("unable to find container associated to the pod %s/%s: %w", namespace, podName, err)
	}

	return adapter.cli.ContainerLogs(ctx, container.ID, types.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Timestamps: opts.Timestamps,
		Follow:     opts.Follow,
		Tail:       opts.Tail,
	})
}

func (adapter *KubeDockerAdapter) GetPodTable(ctx context.Context, namespace string) (*metav1.Table, error) {
	podList, err := adapter.getPodListFromContainers(ctx, namespace)
	if err != nil {
		return &metav1.Table{}, fmt.Errorf("unable to list pods: %w", err)
	}

	return k8s.GenerateTable(&podList)
}

func (adapter *KubeDockerAdapter) ListPods(ctx context.Context, namespace string) (corev1.PodList, error) {
	podList, err := adapter.getPodListFromContainers(ctx, namespace)
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
