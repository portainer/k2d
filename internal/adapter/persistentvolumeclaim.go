package adapter

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/errdefs"
	"github.com/portainer/k2d/internal/adapter/naming"
	k2dtypes "github.com/portainer/k2d/internal/adapter/types"
	"github.com/portainer/k2d/internal/k8s"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubernetes/pkg/apis/core"
)

// TODO: use function comment instead of code comment
// The logic is quite simple, when we create a new PVC we check whether a volume exists.
// The volume name is defined by the PVC name and namespace if the volumeName property is not set on the PVC.
// Otherwise, the volumeName is set to the volumeName property.
// If the volume does not exist, we create it.
func (adapter *KubeDockerAdapter) CreatePersistentVolumeClaim(ctx context.Context, persistentVolumeClaim *corev1.PersistentVolumeClaim) error {
	// if volume name is set, then set the volumeName to the volume name
	// else set the volumeName as a combination of the PVC name and namespace

	volumeName := naming.BuildPersistentVolumeName(persistentVolumeClaim.Name, persistentVolumeClaim.Namespace)
	if persistentVolumeClaim.Spec.VolumeName != "" {
		volumeName = persistentVolumeClaim.Spec.VolumeName
	}

	// check if the volume already exists
	// this is to ensure that we don't create a volume if it already exists
	_, err := adapter.cli.VolumeInspect(ctx, volumeName)

	// TODO: The condition is also not valid, any non nil, non not found error will be caught here
	if !errdefs.IsNotFound(err) {
		// the volume already exists. Update the PVC with the volume name
		// this is a static assignment of a volume to a PVC.

		// TODO: I'm not sure I understand this case. If the volume already exists, why do we need to update the PVC?
		// This property is likely to be set already
		persistentVolumeClaim.Spec.VolumeName = volumeName
	} else if errdefs.IsNotFound(err) {
		// the volume does not exist. Create it
		// this is a dynamic assignment of a volume to a PVC
		_, err = adapter.cli.VolumeCreate(ctx, volume.CreateOptions{
			Name:   volumeName,
			Driver: "local",
			Labels: map[string]string{
				k2dtypes.PersistentVolumeNameLabelKey:      volumeName,
				k2dtypes.PersistentVolumeClaimNameLabelKey: persistentVolumeClaim.Name,
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
				k2dtypes.NamespaceNameLabelKey:                          persistentVolumeClaim.Namespace,
				k2dtypes.PersistentVolumeNameLabelKey:                   volumeName,
				k2dtypes.PersistentVolumeClaimNameLabelKey:              persistentVolumeClaim.Name,
				k2dtypes.PersistentVolumeClaimLastAppliedConfigLabelKey: persistentVolumeClaim.ObjectMeta.Annotations["kubectl.kubernetes.io/last-applied-configuration"],
			},
		},
		// TODO: there is a way to do a bit of optimization here
		// an empty configmap should work but the disk backend store needs to be updated to support empty configmaps
		Data: map[string]string{
			"persistentVolumeClaim": persistentVolumeClaim.Name,
		},
	}

	err = adapter.CreateSystemConfigMap(pvcConfigMap)
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
		return nil, fmt.Errorf("unable to get persistent volume claim: %w", err)
	}

	// TODO: review the updatePersistentVolumeClaimFromVolume function / pattern
	// if possible, review the entire converter pattern
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

	// TODO: review the UpdateConfigMapToPersistentVolumeClaim function / pattern
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
	// configMaps, err := adapter.ListConfigMaps(namespaceName)
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
		// persistentVolumeClaimName := configMap.Labels[k2dtypes.PersistentVolumeClaimNameLabelKey]
		namespace := configMap.Labels[k2dtypes.NamespaceNameLabelKey]

		// TODO: not sure about this condition, seems to me that it is always set
		// if persistentVolumeClaimName != "" {
		if namespaceName == "" || namespace == namespaceName {
			persistentvolumeClaimLastAppliedConfigLabelKey := configMap.Labels[k2dtypes.PersistentVolumeClaimLastAppliedConfigLabelKey]

			if persistentvolumeClaimLastAppliedConfigLabelKey != "" {
				persistentVolumeClaim, err := adapter.updatePersistentVolumeClaimFromVolume(persistentvolumeClaimLastAppliedConfigLabelKey, &configMap)
				if err != nil {
					return core.PersistentVolumeClaimList{}, fmt.Errorf("unable to update persistent volume claim from volume: %w", err)
				}

				persistentVolumeClaims.Items = append(persistentVolumeClaims.Items, *persistentVolumeClaim)
			}
		}
	}

	return persistentVolumeClaims, nil
}
