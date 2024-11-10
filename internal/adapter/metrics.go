package adapter

import (
	"context"
	"fmt"

	"github.com/portainer/k2d/internal/adapter/converter"
	metricsv1beta1 "k8s.io/metrics/pkg/apis/metrics/v1beta1"
)

// GetPodMetrics retrieves the container stats metrics of a container associated to a pod
// and converts them to pod metrics.
func (adapter *KubeDockerAdapter) GetPodMetrics(ctx context.Context, namespace, podName string) (*metricsv1beta1.PodMetrics, error) {
	adapter.logger.Debugf("getting pod metrics for the pod %s in the namespace %s", podName, namespace)
	container, err := adapter.findContainerFromPodAndNamespace(ctx, podName, namespace)
	if err != nil {
		return nil, fmt.Errorf("unable to find container associated to the pod %s/%s: %w", namespace, podName, err)
	}

	containerStats, err := adapter.cli.ContainerStats(ctx, container.Names[0], false)
	if err != nil {
		return nil, err
	}

	return converter.ConvertContainerStatsToPodMetrics(namespace, podName, container.Names[0], &containerStats)
}
