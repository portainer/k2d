package adapter

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/docker/docker/api/types/volume"
	"github.com/portainer/k2d/internal/adapter/naming"
	k2dtypes "github.com/portainer/k2d/internal/adapter/types"
	"github.com/portainer/k2d/internal/k8s"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubernetes/pkg/apis/core"
)

// CreatePersistentVolumeClaim handles the creation or assignment of a Docker volume for a Kubernetes PersistentVolumeClaim (PVC).
//
// Parameters:
//   - ctx:                      A context for managing the lifetime of the request.
//   - persistentVolumeClaim:    A pointer to a Kubernetes PersistentVolumeClaim object that describes the claim.
//
// Returns:
// - An error if any step in the creation or assignment process fails.
//
// Behavior:
//
//   - Static Volume Assignment:
//     If the `Spec.VolumeName` field of the PVC is not empty, the function assumes that this is a static assignment of an existing Docker volume to a PVC.
//     1. Inspects the Docker volume to ensure it exists.
//     2. If the volume does not exist, returns an error.
//
//   - Dynamic Volume Creation:
//     If the `Spec.VolumeName` field of the PVC is empty, the function creates a new Docker volume.
//     1. Generates a dynamic name for the Docker volume based on the PVC name and namespace.
//     2. Creates a new Docker volume with the generated name.
//     3. Labels the Docker volume to identify it as a k2d-managed volume. (See `k2dtypes.PersistentVolumeNameLabelKey`)
//
//   - Helm-managed PVCs:
//     If the PVC has a label "app.kubernetes.io/managed-by" set to "Helm", it serializes the PVC and stores it in an annotation for later use.
//
//   - ConfigMap Creation:
//     Creates a ConfigMap that represents system-level information for the PVC, which includes:
//     1. The namespace of the PVC.
//     2. The name of the Docker volume.
//     3. The name of the PVC.
//     4. The last applied configuration of the PVC
func (adapter *KubeDockerAdapter) CreatePersistentVolumeClaim(ctx context.Context, persistentVolumeClaim *corev1.PersistentVolumeClaim) error {
	var volumeName string

	if persistentVolumeClaim.Spec.VolumeName != "" {
		volumeName = persistentVolumeClaim.Spec.VolumeName
		adapter.logger.Debugf("using existing persistent volume %s for the requested persistent volume claim", volumeName)

		_, err := adapter.cli.VolumeInspect(ctx, volumeName)
		if err != nil {
			return fmt.Errorf("unable to find volume %s: %w", volumeName, err)
		}
	} else {
		volumeName = naming.BuildPersistentVolumeName(persistentVolumeClaim.Name, persistentVolumeClaim.Namespace)
		adapter.logger.Debugf("creating persistent volume %s for the requested persistent volume claim", volumeName)

		_, err := adapter.cli.VolumeCreate(ctx, volume.CreateOptions{
			Name:   volumeName,
			Driver: "local",
			Labels: map[string]string{
				k2dtypes.PersistentVolumeNameLabelKey: volumeName,
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

	pvcConfigMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name: naming.BuildPVCSystemConfigMapName(persistentVolumeClaim.Name, persistentVolumeClaim.Namespace),
			Labels: map[string]string{
				k2dtypes.NamespaceNameLabelKey:             persistentVolumeClaim.Namespace,
				k2dtypes.PersistentVolumeNameLabelKey:      volumeName,
				k2dtypes.PersistentVolumeClaimNameLabelKey: persistentVolumeClaim.Name,
				k2dtypes.LastAppliedConfigLabelKey:         persistentVolumeClaim.ObjectMeta.Annotations["kubectl.kubernetes.io/last-applied-configuration"],
			},
		},
	}
	err := adapter.CreateSystemConfigMap(pvcConfigMap)
	if err != nil {
		return fmt.Errorf("unable to create system configmap for persistent volume claim: %w", err)
	}

	return nil
}

func (adapter *KubeDockerAdapter) DeletePersistentVolumeClaim(ctx context.Context, persistentVolumeClaimName string, namespaceName string) error {
	pvcName := naming.BuildPVCSystemConfigMapName(persistentVolumeClaimName, namespaceName)
	err := adapter.DeleteSystemConfigMap(pvcName)
	if err != nil {
		return fmt.Errorf("unable to delete persistent volume claim: %w", err)
	}

	return nil
}

func (adapter *KubeDockerAdapter) GetPersistentVolumeClaim(ctx context.Context, persistentVolumeClaimName string, namespaceName string) (*corev1.PersistentVolumeClaim, error) {
	pvcName := naming.BuildPVCSystemConfigMapName(persistentVolumeClaimName, namespaceName)
	persistentVolumeClaimConfigMap, err := adapter.GetSystemConfigMap(pvcName)
	if err != nil {
		return nil, fmt.Errorf("unable to get the system configmap associated to the persistent volume claim: %w", err)
	}

	persistentVolumeClaim, err := adapter.updatePersistentVolumeClaimFromVolume(persistentVolumeClaimConfigMap.Labels[k2dtypes.LastAppliedConfigLabelKey], persistentVolumeClaimConfigMap)
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

func (adapter *KubeDockerAdapter) ListPersistentVolumeClaims(ctx context.Context, namespaceName string) (corev1.PersistentVolumeClaimList, error) {
	persistentVolumeClaims, err := adapter.listPersistentVolumeClaims(ctx, namespaceName)
	if err != nil {
		return corev1.PersistentVolumeClaimList{}, fmt.Errorf("unable to list persistent volume claims: %w", err)
	}

	versionedPersistentVolumeClaimList := corev1.PersistentVolumeClaimList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "PersistentVolumeClaimList",
			APIVersion: "v1",
		},
	}

	err = adapter.ConvertK8SResource(&persistentVolumeClaims, &versionedPersistentVolumeClaimList)
	if err != nil {
		return corev1.PersistentVolumeClaimList{}, fmt.Errorf("unable to convert internal PersistentVolumeClaimList to versioned PersistentVolumeClaimList: %w", err)
	}

	return versionedPersistentVolumeClaimList, nil
}

func (adapter *KubeDockerAdapter) GetPersistentVolumeClaimTable(ctx context.Context, namespaceName string) (*metav1.Table, error) {
	persistentVolumeClaims, err := adapter.listPersistentVolumeClaims(ctx, namespaceName)
	if err != nil {
		return &metav1.Table{}, fmt.Errorf("unable to list persistent volume claim: %w", err)
	}

	return k8s.GenerateTable(&persistentVolumeClaims)
}

func (adapter *KubeDockerAdapter) listPersistentVolumeClaims(ctx context.Context, namespaceName string) (core.PersistentVolumeClaimList, error) {
	configMaps, err := adapter.ListSystemConfigMaps()
	if err != nil {
		return core.PersistentVolumeClaimList{}, fmt.Errorf("unable to list configmaps: %w", err)
	}

	persistentVolumeClaims := core.PersistentVolumeClaimList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "PersistentVolumeClaimList",
			APIVersion: "v1",
		},
	}

	for _, configMap := range configMaps.Items {
		namespace := configMap.Labels[k2dtypes.NamespaceNameLabelKey]

		if namespaceName == "" || namespace == namespaceName {
			pvcLastAppliedConfig := configMap.Labels[k2dtypes.LastAppliedConfigLabelKey]

			if pvcLastAppliedConfig != "" {
				persistentVolumeClaim, err := adapter.updatePersistentVolumeClaimFromVolume(pvcLastAppliedConfig, &configMap)
				if err != nil {
					return core.PersistentVolumeClaimList{}, fmt.Errorf("unable to update persistent volume claim from volume: %w", err)
				}
				persistentVolumeClaims.Items = append(persistentVolumeClaims.Items, *persistentVolumeClaim)
			}
		}
	}

	return persistentVolumeClaims, nil
}
