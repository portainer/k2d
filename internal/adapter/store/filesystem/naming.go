package filesystem

import "fmt"

// Each key of a configmap is stored in a separate file using the following naming convention:
// [namespace]-[configmap-name]-k2dcm-[key]
func buildConfigMapFilePrefix(configMapName, namespace string) string {
	return fmt.Sprintf("%s-%s%s", namespace, configMapName, ConfigMapSeparator)
}

// Each configmap has its own metadata file that follows the naming convention below:
// [namespace]-[configmap-name]-k2dcm.metadata
func buildConfigMapMetadataFileName(configMapName, namespace string) string {
	return fmt.Sprintf("%s-%s-k2dcm.metadata", namespace, configMapName)
}

// Each key of a secret is stored in a separate file using the following naming convention:
// [namespace]-[secret-name]-k2dsec-[key]
func buildSecretFilePrefix(secretName, namespace string) string {
	return fmt.Sprintf("%s-%s%s", namespace, secretName, SecretSeparator)
}

// Each secret has its own metadata file that follows the naming convention below:
// [namespace]-[secret-name]-k2dsec.metadata
func buildSecretMetadataFileName(secretName, namespace string) string {
	return fmt.Sprintf("%s-%s-k2dsec.metadata", namespace, secretName)
}
