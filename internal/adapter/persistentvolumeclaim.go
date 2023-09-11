package adapter

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/errdefs"
	adaptererr "github.com/portainer/k2d/internal/adapter/errors"
	"github.com/portainer/k2d/internal/adapter/naming"
	k2dtypes "github.com/portainer/k2d/internal/adapter/types"
	"github.com/portainer/k2d/internal/k8s"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubernetes/pkg/apis/core"
)

func (adapter *KubeDockerAdapter) CreatePersistentVolumeClaim(ctx context.Context, persistentVolumeClaim *corev1.PersistentVolumeClaim) error {
	volumeName := naming.BuildPersistentVolumeName(persistentVolumeClaim.Name, persistentVolumeClaim.Namespace)

	// check if the volume already exists
	// this is to ensure that we don't create a volume if it already exists
	_, err := adapter.cli.VolumeInspect(ctx, volumeName)
	if !errdefs.IsNotFound(err) {
		// the volume already exists. Update the PVC with the volume name
		// this is a static assignment of a volume to a PVC
		persistentVolumeClaim.Spec.VolumeName = volumeName
	} else if errdefs.IsNotFound(err) {
		// the volume does not exist. Create it
		// this is a dynamic assignment of a volume to a PVC
		_, err = adapter.cli.VolumeCreate(ctx, volume.CreateOptions{
			Name:   volumeName,
			Driver: "local",
			Labels: map[string]string{
				k2dtypes.NamespaceLabelKey:        persistentVolumeClaim.Namespace,
				k2dtypes.PersistentVolumeLabelKey: volumeName,
			},
		})

		if err != nil {
			return fmt.Errorf("unable to create a Docker volume for the request persistent volume claim: %w", err)
		}
	}

	if persistentVolumeClaim.Labels["app.kubernetes.io/managed-by"] == "Helm" {
		persistentVolumeClaimData, err := json.Marshal(persistentVolumeClaim)
		if err != nil {
			return fmt.Errorf("unable to marshal deployment: %w", err)
		}
		persistentVolumeClaim.ObjectMeta.Annotations["kubectl.kubernetes.io/last-applied-configuration"] = string(persistentVolumeClaimData)
	}

	err = adapter.CreateConfigMap(&corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      persistentVolumeClaim.Name,
			Namespace: persistentVolumeClaim.Namespace,
			// using annotations instsead of data to store the following values
			// the reason is that when these are stored on a local file system, then ConfigMapSeparator uses the / which is then treated as a folder
			// for instance, namespace.k2d.io/name
			// todo: discuss if this is the best way to store this information as a metadata only
			Labels: map[string]string{
				k2dtypes.NamespaceLabelKey:                              persistentVolumeClaim.Namespace,
				k2dtypes.PersistentVolumeLabelKey:                       volumeName,
				k2dtypes.PersistentVolumeClaimLabelKey:                  persistentVolumeClaim.Name,
				k2dtypes.PersistentVolumeClaimLastAppliedConfigLabelKey: persistentVolumeClaim.ObjectMeta.Annotations["kubectl.kubernetes.io/last-applied-configuration"],
			},
		},
		Data: map[string]string{
			"persistentVolumeClaim": persistentVolumeClaim.Name,
		},
	})

	if err != nil {
		return fmt.Errorf("unable to create configmap for persistent volume claim: %w", err)
	}

	return nil
}

func (adapter *KubeDockerAdapter) DeletePersistentVolumeClaim(ctx context.Context, persistentVolumeClaimName string, namespaceName string) error {
	// this needs updating by deleting a configMap instead of a volume
	err := adapter.DeleteConfigMap(persistentVolumeClaimName, namespaceName)
	if err != nil {
		return fmt.Errorf("unable to delete persistent volume claim: %w", err)
	}

	return nil
}

func (adapter *KubeDockerAdapter) GetPersistentVolumeClaim(ctx context.Context, persistentVolumeClaimName string, namespaceName string) (*corev1.PersistentVolumeClaim, error) {
	persistentVolumeClaimConfigMap, err := adapter.GetConfigMap(persistentVolumeClaimName, namespaceName)

	if err != nil {
		return nil, fmt.Errorf("unable to get persistent volume claim: %w", err)
	}

	if persistentVolumeClaimConfigMap == nil {
		return nil, adaptererr.ErrResourceNotFound
	}

	persistentVolumeClaim, err := adapter.updatePersistentVolumeClaimFromVolume(persistentVolumeClaimConfigMap.Labels[k2dtypes.PersistentVolumeClaimLastAppliedConfigLabelKey], persistentVolumeClaimConfigMap)
	if err != nil {
		return nil, fmt.Errorf("unable to update persistent volume claim from volume: %w", err)
	}

	versionedpersistentVolumeClaim := corev1.PersistentVolumeClaim{
		TypeMeta: metav1.TypeMeta{
			Kind:       "PersistentVolumeClaim",
			APIVersion: "v1",
		},
	}

	err = adapter.ConvertK8SResource(persistentVolumeClaim, &versionedpersistentVolumeClaim)
	if err != nil {
		return nil, fmt.Errorf("unable to convert internal object to versioned object: %w", err)
	}

	return &versionedpersistentVolumeClaim, nil
}

func (adapter *KubeDockerAdapter) updatePersistentVolumeClaimFromVolume(persistentVolumeClaimData string, configMap *corev1.ConfigMap) (*core.PersistentVolumeClaim, error) {
	versionedPersistentVolumeClaim := &corev1.PersistentVolumeClaim{}

	err := json.Unmarshal([]byte(persistentVolumeClaimData), &versionedPersistentVolumeClaim)
	if err != nil {
		return nil, fmt.Errorf("unable to unmarshal versioned service: %w", err)
	}

	persistentVolumeClaim := core.PersistentVolumeClaim{}
	err = adapter.ConvertK8SResource(versionedPersistentVolumeClaim, &persistentVolumeClaim)
	if err != nil {
		return nil, fmt.Errorf("unable to convert internal object to versioned object: %w", err)
	}

	err = adapter.converter.UpdateConfigMapToPersistentVolumeClaim(&persistentVolumeClaim, configMap)
	if err != nil {
		return nil, fmt.Errorf("unable to convert Docker volume to PersistentVolumeClaim: %w", err)
	}

	return &persistentVolumeClaim, nil
}

func (adapter *KubeDockerAdapter) ListPersistentVolumeClaims(ctx context.Context, namespaceName string) (core.PersistentVolumeClaimList, error) {
	persistentVolumeClaims, err := adapter.listPersistentVolumeClaims(ctx, namespaceName)
	if err != nil {
		return core.PersistentVolumeClaimList{}, fmt.Errorf("unable to list persistent volume claims: %w", err)
	}

	return *persistentVolumeClaims, nil
}

func (adapter *KubeDockerAdapter) GetPersistentVolumeClaimTable(ctx context.Context, namespaceName string) (*metav1.Table, error) {
	persistentVolumeClaims, err := adapter.listPersistentVolumeClaims(ctx, namespaceName)
	if err != nil {
		return &metav1.Table{}, fmt.Errorf("unable to list persistent volume claim: %w", err)
	}

	return k8s.GenerateTable(persistentVolumeClaims)
}

func (adapter *KubeDockerAdapter) listPersistentVolumeClaims(ctx context.Context, namespaceName string) (*core.PersistentVolumeClaimList, error) {
	configMaps, err := adapter.ListConfigMaps(namespaceName)
	if err != nil {
		return nil, fmt.Errorf("unable to list configmaps: %w", err)
	}

	persistentVolumeClaims := core.PersistentVolumeClaimList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "PersistentVolumeClaimList",
			APIVersion: "v1",
		},
	}

	for _, configMap := range configMaps.Items {
		persistentVolumeClaimConfigMap, err := adapter.GetConfigMap(configMap.Labels[k2dtypes.PersistentVolumeClaimLabelKey], namespaceName)

		if err != nil {
			return nil, fmt.Errorf("unable to get persistent volume claim: %w", err)
		}

		if persistentVolumeClaimConfigMap == nil {
			continue
		}

		if persistentVolumeClaimConfigMap.Labels[k2dtypes.PersistentVolumeClaimLastAppliedConfigLabelKey] != "" {

			persistentVolumeClaim, err := adapter.updatePersistentVolumeClaimFromVolume(persistentVolumeClaimConfigMap.Labels[k2dtypes.PersistentVolumeClaimLastAppliedConfigLabelKey], persistentVolumeClaimConfigMap)
			if err != nil {
				return nil, fmt.Errorf("unable to update persistent volume claim from volume: %w", err)
			}

			persistentVolumeClaims.Items = append(persistentVolumeClaims.Items, *persistentVolumeClaim)
		}
	}

	return &persistentVolumeClaims, nil
}
