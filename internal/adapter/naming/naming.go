package naming

import (
	"fmt"
	"strings"
)

// Each container is named using the following format:
// [namespace]-[container-name]
func BuildContainerName(containerName, namespace string) string {
	containerName = strings.TrimPrefix(containerName, "/")
	return fmt.Sprintf("%s-%s", namespace, containerName)
}

// Each network is named using the following format:
// k2d-[namespace]
func BuildNetworkName(namespace string) string {
	return fmt.Sprintf("k2d-%s", namespace)
}

// Each persistentVolume is named using the following format:
// k2d-pv-[namespace]-[volume-name]
func BuildPersistentVolumeName(volumeName string, namespace string) string {
	return fmt.Sprintf("k2d-pv-%s-%s", namespace, volumeName)
}
