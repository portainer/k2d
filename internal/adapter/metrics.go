package adapter

import (
	"context"
	"encoding/json"
	"io"
	"strconv"

	"github.com/portainer/k2d/internal/adapter/naming"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	metricsv1beta1 "k8s.io/metrics/pkg/apis/metrics/v1beta1"
)

func (adapter *KubeDockerAdapter) GetPodMetrics(ctx context.Context, podName string, namespace string) (*metricsv1beta1.PodMetrics, error) {
	containerName := naming.BuildContainerName(podName, namespace)
	containerMetrics, err := adapter.cli.ContainerStats(ctx, containerName, false)
	if err != nil {
		return nil, err
	}

	containerMetricsBody, err := io.ReadAll(containerMetrics.Body)
	if err != nil {
		return nil, err
	}

	var metricsBody map[string]any
	err = json.Unmarshal(containerMetricsBody, &metricsBody)
	if err != nil {
		return nil, err
	}

	// metricsBody["cpu_stats"].(map[string]any)["cpu_usage"].(map[string]any)["total_usage"].(float64)
	// memoryStatsString :=

	// resource list
	containerResourceList := v1.ResourceList{
		v1.ResourceCPU:    resource.MustParse(strconv.FormatFloat(metricsBody["cpu_stats"].(map[string]any)["cpu_usage"].(map[string]any)["total_usage"].(float64), 'f', 0, 64)),
		v1.ResourceMemory: resource.MustParse(strconv.FormatFloat(metricsBody["memory_stats"].(map[string]any)["usage"].(float64), 'f', 0, 64)),
	}

	// construct a v1beta1 metrics API
	podMetrics := &metricsv1beta1.PodMetrics{
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
				Name:  "node-red",
				Usage: containerResourceList,
			},
		},
	}

	return podMetrics, nil
}
