package adapter

import (
	"context"
	"fmt"

	"github.com/portainer/k2d/pkg/filesystem"
	"k8s.io/apimachinery/pkg/labels"
)

// ExecuteResetRoutine performs a cleanup routine that removes all k2d resources from the host.
// This function is intended to be used as a "reset mode" operation for cleaning up any resources
// managed by k2d on the host.
//
// Parameters:
//   - ctx context.Context: The context for carrying out the reset routine.
//
// Returns:
//   - error: Returns an error if any of the resource removal operations fail.
//
// Steps:
//  1. Removes all workload resources (like deployments, pods) by invoking removeAllWorkloads.
//  2. Removes all Persistent Volumes and Persistent Volume Claims by invoking removeAllPersistentVolumeAndClaims.
//  3. Removes all ConfigMaps and Secrets by invoking removeAllConfigMapsAndSecrets.
//  4. Removes all namespaces by invoking removeAllNamespaces.
func (adapter *KubeDockerAdapter) ExecuteResetRoutine(ctx context.Context, k2dDataPath string) error {
	adapter.logger.Infoln("reset mode enabled, removing all k2d resources on this host")

	err := adapter.removeAllWorkloads(ctx)
	if err != nil {
		return fmt.Errorf("unable to remove workloads: %w", err)
	}

	err = adapter.removeAllPersistentVolumeAndClaims(ctx)
	if err != nil {
		return fmt.Errorf("unable to remove persistent volumes and persistent volume claims: %w", err)
	}

	err = adapter.removeAllConfigMapsAndSecrets(ctx)
	if err != nil {
		return fmt.Errorf("unable to remove configmaps and secrets: %w", err)
	}

	err = adapter.removeAllNamespaces(ctx)
	if err != nil {
		return fmt.Errorf("unable to remove namespaces: %w", err)
	}

	adapter.logger.Infoln("removing k2d data directory content...")

	err = filesystem.RemoveAllContent(k2dDataPath)
	if err != nil {
		return fmt.Errorf("unable to remove k2d data directory content: %w", err)
	}

	adapter.logger.Infoln("reset routine completed")
	return nil
}

func (adapter *KubeDockerAdapter) removeAllWorkloads(ctx context.Context) error {
	adapter.logger.Infoln("removing all workloads (deployments, pods)...")

	deployments, err := adapter.ListDeployments(ctx, "")
	if err != nil {
		return fmt.Errorf("unable to list deployments: %w", err)
	}

	for _, deployment := range deployments.Items {
		adapter.logger.Infof("removing deployment %s/%s", deployment.Namespace, deployment.Name)
		adapter.DeleteContainer(ctx, deployment.Name, deployment.Namespace)
	}

	pods, err := adapter.ListPods(ctx, "")
	if err != nil {
		return fmt.Errorf("unable to list pods: %w", err)
	}

	for _, pod := range pods.Items {
		adapter.logger.Infof("removing pod %s/%s", pod.Namespace, pod.Name)
		adapter.DeleteContainer(ctx, pod.Name, pod.Namespace)
	}

	return nil
}

func (adapter *KubeDockerAdapter) removeAllPersistentVolumeAndClaims(ctx context.Context) error {
	adapter.logger.Infoln("removing all persistent volumes and persistent volume claims...")

	persistentVolumes, err := adapter.ListPersistentVolumes(ctx)
	if err != nil {
		return fmt.Errorf("unable to list persistent volumes: %w", err)
	}

	for _, persistentVolume := range persistentVolumes.Items {
		adapter.logger.Infof("removing persistent volume %s", persistentVolume.Name)

		err = adapter.DeletePersistentVolume(ctx, persistentVolume.Name)
		if err != nil {
			adapter.logger.Warnf("unable to remove persistent volume %s: %s", persistentVolume.Name, err)
		}
	}

	persistentVolumeClaims, err := adapter.ListPersistentVolumeClaims(ctx, "")
	if err != nil {
		return fmt.Errorf("unable to list persistent volume claims: %w", err)
	}

	for _, persistentVolumeClaim := range persistentVolumeClaims.Items {
		adapter.logger.Infof("removing persistent volume claim %s/%s", persistentVolumeClaim.Namespace, persistentVolumeClaim.Name)

		err = adapter.DeletePersistentVolumeClaim(ctx, persistentVolumeClaim.Name, persistentVolumeClaim.Namespace)
		if err != nil {
			adapter.logger.Warnf("unable to remove persistent volume claim %s/%s: %s", persistentVolumeClaim.Namespace, persistentVolumeClaim.Name, err)
		}
	}

	return nil
}

func (adapter *KubeDockerAdapter) removeAllConfigMapsAndSecrets(ctx context.Context) error {
	adapter.logger.Infoln("removing all configmaps...")

	configMaps, err := adapter.ListConfigMaps("")
	if err != nil {
		return fmt.Errorf("unable to list configmaps: %w", err)
	}

	for _, configMap := range configMaps.Items {
		adapter.logger.Infof("removing configmap %s/%s", configMap.Namespace, configMap.Name)

		err = adapter.DeleteConfigMap(configMap.Name, configMap.Namespace)
		if err != nil {
			adapter.logger.Warnf("unable to remove configmap %s/%s: %s", configMap.Namespace, configMap.Name, err)
		}
	}

	adapter.logger.Infoln("removing all secrets...")

	secrets, err := adapter.ListSecrets("", labels.NewSelector())
	if err != nil {
		return fmt.Errorf("unable to list secrets: %w", err)
	}

	for _, secret := range secrets.Items {
		adapter.logger.Infof("removing secret %s/%s", secret.Namespace, secret.Name)

		err = adapter.DeleteSecret(secret.Name, secret.Namespace)
		if err != nil {
			adapter.logger.Warnf("unable to remove secret %s/%s: %s", secret.Namespace, secret.Name, err)
		}
	}

	return nil
}

func (adapter *KubeDockerAdapter) removeAllNamespaces(ctx context.Context) error {
	adapter.logger.Infoln("removing all namespaces...")

	namespaces, err := adapter.ListNamespaces(ctx)
	if err != nil {
		return fmt.Errorf("unable to list namespaces: %w", err)
	}

	for _, namespace := range namespaces.Items {
		adapter.logger.Infof("removing namespace %s", namespace.Name)

		err = adapter.DeleteNamespace(ctx, namespace.Name)
		if err != nil {
			adapter.logger.Warnf("unable to remove namespace %s: %s", namespace.Name, err)
		}
	}

	return nil
}
