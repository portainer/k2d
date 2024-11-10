package converter

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/docker/docker/api/types/container"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	metricsv1beta1 "k8s.io/metrics/pkg/apis/metrics/v1beta1"
)

// ConvertContainerStatsToPodMetrics converts Docker container stats to pod metrics.
// the CPU calcaulation is based on the following:
// https://github.com/docker/cli/blob/master/cli/command/container/stats_helpers.go#L166
// whereas the memory calculation is simply the current memory usage.
func ConvertContainerStatsToPodMetrics(namespace, podName, containerName string, containerStatsResponse *container.StatsResponseReader) (*metricsv1beta1.PodMetrics, error) {
	containerMetricsBody, err := io.ReadAll(containerStatsResponse.Body)
	if err != nil {
		return nil, err
	}
	defer containerStatsResponse.Body.Close()

	containerStats := container.Stats{}
	err = json.Unmarshal(containerMetricsBody, &containerStats)
	if err != nil {
		return nil, err
	}

	return &metricsv1beta1.PodMetrics{
		TypeMeta: metav1.TypeMeta{
			Kind:       "PodMetrics",
			APIVersion: "metrics.k8s.io/v1beta1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      podName,
			Namespace: namespace,
		},
		Timestamp: metav1.Time{},
		Window:    metav1.Duration{},
		Containers: []metricsv1beta1.ContainerMetrics{
			{
				Name: containerName,
				Usage: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse(fmt.Sprintf("%f", calculateCPUUsage(&containerStats))),
					corev1.ResourceMemory: resource.MustParse(fmt.Sprintf("%d", containerStats.MemoryStats.Usage)),
				},
			},
		},
	}, nil
}

func calculateCPUUsage(containerStats *container.Stats) float64 {
	cpuDelta := float64(containerStats.CPUStats.CPUUsage.TotalUsage) - float64(containerStats.PreCPUStats.CPUUsage.TotalUsage)
	systemDelta := float64(containerStats.CPUStats.SystemUsage) - float64(containerStats.PreCPUStats.SystemUsage)
	onlineCPUs := float64(containerStats.CPUStats.OnlineCPUs)

	if onlineCPUs == 0.0 {
		onlineCPUs = float64(len(containerStats.CPUStats.CPUUsage.PercpuUsage))
	}

	if systemDelta > 0.0 && cpuDelta > 0.0 {
		return (cpuDelta / systemDelta) * onlineCPUs
	}

	return 0
}
