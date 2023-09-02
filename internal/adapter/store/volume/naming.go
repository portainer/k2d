package volume

import (
	"fmt"
	"strings"
)

// Each configmap is stored as a Docker volume using the following naming convention:
// configmap-[namespace]-[configmap-name]
func buildConfigMapVolumeName(configMapName, namespace string) string {
	return fmt.Sprintf("%s%s-%s", ConfigMapVolumePrefix, namespace, configMapName)
}

// Each secret is stored as a Docker volume using the following naming convention:
// secret-[namespace]-[secret-name]
func buildSecretVolumeName(configMapName, namespace string) string {
	return fmt.Sprintf("%s%s-%s", SecretVolumePrefix, namespace, configMapName)
}

// Returns [configmap-name] from configmap-[namespace]-[configmap-name]
func getConfigMapNameFromVolumeName(volumeName, namespace string) string {
	return strings.TrimPrefix(volumeName, fmt.Sprintf("%s%s-", ConfigMapVolumePrefix, namespace))
}

// Returns [secret-name] from secret-[namespace]-[secret-name]
func getSecretNameFromVolumeName(volumeName, namespace string) string {
	return strings.TrimPrefix(volumeName, fmt.Sprintf("%s%s-", SecretVolumePrefix, namespace))
}
