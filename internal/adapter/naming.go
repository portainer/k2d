package adapter

import (
	"fmt"
	"strings"
)

// Each container is named using the following format:
// [namespace]-[container-name]
func buildContainerName(containerName, namespace string) string {
	containerName = strings.TrimPrefix(containerName, "/")
	return fmt.Sprintf("%s-%s", namespace, containerName)
}
