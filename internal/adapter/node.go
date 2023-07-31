package adapter

import (
	"context"
	"fmt"

	"k8s.io/kubernetes/pkg/apis/core"
)

func (adapter *KubeDockerAdapter) ListNodes(ctx context.Context) (core.NodeList, error) {
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
