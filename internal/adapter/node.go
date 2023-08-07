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

func (adapter *KubeDockerAdapter) GetNode(ctx context.Context) (*corev1.Node, error) {
	node, err := adapter.getNode(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to get node: %w", err)
	}

	versionedNode := corev1.Node{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Node",
			APIVersion: "v1",
		},
	}

	err = adapter.ConvertK8SResource(node, &versionedNode)
	if err != nil {
		return nil, fmt.Errorf("unable to convert internal object to versioned object: %w", err)
	}

	return &versionedNode, nil
}

func (adapter *KubeDockerAdapter) GetNodeTable(ctx context.Context) (*metav1.Table, error) {
	nodeList, err := adapter.listNodes(ctx)
	if err != nil {
		return &metav1.Table{}, fmt.Errorf("unable to list nodes: %w", err)
	}

	return k8s.GenerateTable(&nodeList)
}

func (adapter *KubeDockerAdapter) getNode(ctx context.Context) (*core.Node, error) {
	info, err := adapter.cli.Info(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve docker server info: %w", err)
	}

	version, err := adapter.cli.ServerVersion(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve docker server version: %w", err)
	}

	node := adapter.converter.ConvertInfoVersionToNode(info, version, adapter.startTime)
	return &node, nil
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

	return core.NodeList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "NodeList",
			APIVersion: "v1",
		},
		Items: []core.Node{
			adapter.converter.ConvertInfoVersionToNode(info, version, adapter.startTime),
		},
	}, nil
}
