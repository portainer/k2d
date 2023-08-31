package filters

import (
	"fmt"

	"github.com/docker/docker/api/types/filters"
	k2dtypes "github.com/portainer/k2d/internal/adapter/types"
)

// NamespaceFilter creates a Docker filter argument for a given Kubernetes namespace.
// This function uses the k2dtypes.NamespaceLabelKey constant as the base label key to filter Docker resources.
//
// Parameters:
//   - namespace: The Kubernetes namespace to filter by. If this is an empty string, the function will
//     generate a filter that matches resources with the k2dtypes.NamespaceLabelKey label,
//     regardless of its value. This implies that the filter will match resources in all namespaces.
//
// Returns:
// - filters.Args: A Docker filter object that can be used to filter Docker API calls based on the namespace label.
//
// Usage Example:
//
//	filter := NamespaceFilter("default")
//	// Now 'filter' can be used in Docker API calls to filter resources in the 'default' Kubernetes namespace.
func NamespaceFilter(namespace string) filters.Args {
	label := k2dtypes.NamespaceLabelKey
	if namespace != "" {
		label = fmt.Sprintf("%s=%s", k2dtypes.NamespaceLabelKey, namespace)
	}

	return filters.NewArgs(filters.Arg("label", label))
}
