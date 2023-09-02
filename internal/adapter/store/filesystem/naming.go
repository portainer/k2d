package filesystem

import (
	"fmt"
	"strings"
)

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

// Returns [configmap-name] from [namespace]-[configmap-name]
func getConfigMapNameFromNamespacedConfigMapName(namespacedConfigMapName, namespace string) string {
	return strings.TrimPrefix(namespacedConfigMapName, namespace+"-")
}

// Returns [namespace]-[configmap-name] from [namespace]-[configmap-name]-k2dcm.metadata
func getNamespacedConfigMapNameFromMetadataFileName(fileName string) string {
	return strings.TrimSuffix(fileName, "-k2dcm.metadata")
}

// Returns [namespace]-[configmap-name] and [key] from [namespace]-[configmap-name]-k2dcm-[key]
// or an error if the file name is not matching the expected format
func getNamespacedConfigMapNameAndKeyFromFileName(fileName string) (string, string, error) {
	split := strings.SplitN(fileName, ConfigMapSeparator, 2)

	if len(split) != 2 {
		return "", "", fmt.Errorf("invalid secret file name: %s", fileName)
	}

	return split[0], split[1], nil
}

// Returns [namespace]-[secret-name] and [key] from [namespace]-[secret-name]-k2dsec-[key]
// or an error if the file name is not matching the expected format
func getNamespacedSecretNameAndKeyFromFileName(fileName string) (string, string, error) {
	split := strings.SplitN(fileName, SecretSeparator, 2)

	if len(split) != 2 {
		return "", "", fmt.Errorf("invalid secret file name: %s", fileName)
	}

	return split[0], split[1], nil
}

// Returns [namespace]-[secret-name] from [namespace]-[secret-name]-k2dsec.metadata
func getNamespacedSecretNameFromMetadataFileName(fileName string) string {
	return strings.TrimSuffix(fileName, "-k2dsec.metadata")
}

// Returns [secret-name] from [namespace]-[secret-name]
func getSecretNameFromNamespacedSecretName(namespacedSecretName, namespace string) string {
	return strings.TrimPrefix(namespacedSecretName, namespace+"-")
}
