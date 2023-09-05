package volume

import (
	"fmt"
	"strings"
)

const (
	// ConfigMapVolumePrefix is the prefix used to name volumes associated to ConfigMap resources
	// A prefix is used to avoid clash with Secret volumes
	ConfigMapVolumePrefix = "k2d-configmap-"

	// SecretVolumePrefix is the prefix used to name volumes associated to Secret resources
	// A prefix is used to avoid clash with ConfigMap volumes
	SecretVolumePrefix = "k2d-secret-"
)

// Each configmap is stored as a Docker volume using the following naming convention:
// k2d-configmap-[namespace]-[configmap-name]
func buildConfigMapVolumeName(configMapName, namespace string) string {
	return fmt.Sprintf("%s%s-%s", ConfigMapVolumePrefix, namespace, configMapName)
}

// Each secret is stored as a Docker volume using the following naming convention:
// k2d-secret-[namespace]-[secret-name]
func buildSecretVolumeName(configMapName, namespace string) string {
	return fmt.Sprintf("%s%s-%s", SecretVolumePrefix, namespace, configMapName)
}

// Returns [configmap-name] from k2d-configmap-[namespace]-[configmap-name]
func getConfigMapNameFromVolumeName(volumeName, namespace string) string {
	return strings.TrimPrefix(volumeName, fmt.Sprintf("%s%s-", ConfigMapVolumePrefix, namespace))
}

// Returns [secret-name] from k2d-secret-[namespace]-[secret-name]
func getSecretNameFromVolumeName(volumeName, namespace string) string {
	return strings.TrimPrefix(volumeName, fmt.Sprintf("%s%s-", SecretVolumePrefix, namespace))
}
