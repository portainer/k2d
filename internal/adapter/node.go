package adapter

import (
	"context"
	"fmt"

	"github.com/portainer/k2d/internal/k8s"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubernetes/pkg/apis/core"
)

func (adapter *KubeDockerAdapter) ListNodes(ctx context.Context) (corev1.NodeList, error) {
	nodeList, err := adapter.listNodes(ctx)
	if err != nil {
		return corev1.NodeList{}, fmt.Errorf("unable to list nodes: %w", err)
	}

	versionedNodeList := corev1.NodeList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "NodeList",
			APIVersion: "v1",
		},
	}

	err = adapter.ConvertK8SResource(&nodeList, &versionedNodeList)
	if err != nil {
		return corev1.NodeList{}, fmt.Errorf("unable to convert internal NodeList to versioned NodeList: %w", err)
	}

	return versionedNodeList, nil
}

func (adapter *KubeDockerAdapter) GetNodeTable(ctx context.Context) (*metav1.Table, error) {
	nodeList, err := adapter.listNodes(ctx)
	if err != nil {
		return &metav1.Table{}, fmt.Errorf("unable to list nodes: %w", err)
	}

	return k8s.GenerateTable(&nodeList)
}

func (adapter *KubeDockerAdapter) listNodes(ctx context.Context) (core.NodeList, error) {
	info, err := adapter.cli.Info(ctx)
	if err != nil {
		return core.NodeList{}, fmt.Errorf("unable to retrieve docker server info: %w", err)
	}

	version, err := adapter.cli.ServerVersion(ctx)
	if err != nil {
		return core.NodeList{}, fmt.Errorf("unable to retrieve docker server version: %w", err)
	}

	nodeList := adapter.converter.ConvertInfoVersionToNodes(info, version, adapter.startTime)

	return nodeList, nil
}
