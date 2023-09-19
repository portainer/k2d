package filters

import (
	"fmt"

	"github.com/docker/docker/api/types/filters"
	"github.com/portainer/k2d/internal/adapter/types"
)

// AllDeployments creates a Docker filter argument for Kubernetes Deployments within a given namespace.
// The function filters Docker resources based on the Workload and Namespace labels, specifically for Deployments.
//
// Parameters:
//   - namespace: The Kubernetes namespace to filter by.
//
// Returns:
// - filters.Args: A Docker filter object that can be used to filter Docker API calls based on the namespace and Workload type labels.
//
// Usage Example:
//
//	filter := AllDeployments("default")
//	// Now 'filter' can be used in Docker API calls to filter Deployment resources in the 'default' Kubernetes namespace.
func AllDeployments(namespace string) filters.Args {
	filter := ByNamespace(namespace)
	filter.Add("label", fmt.Sprintf("%s=%s", types.WorkloadLabelKey, types.DeploymentWorkloadType))
	return filter
}

// AllNamespaces creates a Docker filter argument that targets resources labeled with a Kubernetes namespace.
// This function uses the types.NamespaceLabelKey constant as the base label key to filter Docker resources.
//
// Parameters:
//   - None
//
// Returns:
// - filters.Args: A Docker filter object that can be used to filter Docker API calls based on the presence of the namespace label.
//
// Usage Example:
//
//	filter := AllNamespaces()
//	// Now 'filter' can be used in Docker API calls to filter resources that are labeled with any Kubernetes namespace.
func AllNamespaces() filters.Args {
	return filters.NewArgs(filters.Arg("label", types.NamespaceNameLabelKey))
}

// AllServices creates a Docker filter argument to target all Docker resources labeled as services within a specific Kubernetes namespace.
//
// Parameters:
//   - namespace: The Kubernetes namespace to filter by.
//
// Returns:
// - filters.Args: A Docker filter object to be used in Docker API calls to filter resources that are labeled as services within the specified namespace.
//
// Usage Example:
//
//	filter := AllServices("default")
//	// Now 'filter' can be used in Docker API calls to filter service resources in the 'default' Kubernetes namespace.
func AllServices(namespace string) filters.Args {
	filter := ByNamespace(namespace)
	filter.Add("label", types.ServiceNameLabelKey)
	return filter
}

// ByDeployment creates a Docker filter argument for a specific Kubernetes Deployment within a given namespace.
// The function builds upon the DeploymentsFilter by further narrowing down the filter to match a specific Deployment name.
//
// Parameters:
//   - namespace: The Kubernetes namespace to filter by.
//   - deploymentName: The name of the specific Kubernetes Deployment to filter by.
//
// Returns:
// - filters.Args: A Docker filter object that can be used to filter Docker API calls based on the namespace and Deployment name labels.
//
// Usage Example:
//
//	filter := ByDeployment("default", "my-deployment")
//	// Now 'filter' can be used in Docker API calls to filter resources in the 'default' Kubernetes namespace that are part of 'my-deployment'.
func ByDeployment(namespace, deploymentName string) filters.Args {
	filter := AllDeployments(namespace)
	filter.Add("label", fmt.Sprintf("%s=%s", types.WorkloadNameLabelKey, deploymentName))
	return filter
}

// ByNamespace creates a Docker filter argument to target all Docker resources within a specific Kubernetes namespace.
// If an empty string is provided, it will return a filter that targets all namespaces.
//
// Parameters:
//   - namespace: The Kubernetes namespace to filter by, or an empty string for all namespaces.
//
// Returns:
// - filters.Args: A Docker filter object to be used in Docker API calls to filter resources within the specified namespace or all namespaces.
//
// Usage Example:
//
//	filter := ByNamespace("default")
//	// Now 'filter' can be used in Docker API calls to filter resources in the 'default' Kubernetes namespace.
func ByNamespace(namespace string) filters.Args {
	if namespace == "" {
		return AllNamespaces()
	}
	return filters.NewArgs(filters.Arg("label", fmt.Sprintf("%s=%s", types.NamespaceNameLabelKey, namespace)))
}

// ByPod creates a Docker filter argument to target a specific pod within a specific Kubernetes namespace.
//
// Parameters:
//   - namespace: The Kubernetes namespace to filter by.
//   - podName: The name of the pod to filter by.
//
// Returns:
// - filters.Args: A Docker filter object to be used in Docker API calls to filter resources for a specific pod within the specified namespace.
//
// Usage Example:
//
//	filter := ByPod("default", "mypod")
//	// Now 'filter' can be used in Docker API calls to filter resources of pod 'mypod' in the 'default' Kubernetes namespace.
func ByPod(namespace, podName string) filters.Args {
	filter := filters.NewArgs()
	filter.Add("label", fmt.Sprintf("%s=%s", types.NamespaceNameLabelKey, namespace))
	filter.Add("label", fmt.Sprintf("%s=%s", types.WorkloadNameLabelKey, podName))
	return filter
}

// ByService creates a Docker filter argument to target a specific service within a specific Kubernetes namespace.
//
// Parameters:
//   - namespace: The Kubernetes namespace to filter by.
//   - serviceName: The name of the service to filter by.
//
// Returns:
// - filters.Args: A Docker filter object to be used in Docker API calls to filter resources for a specific service within the specified namespace.
//
// Usage Example:
//
//	filter := ByService("default", "myservice")
//	// Now 'filter' can be used in Docker API calls to filter resources of service 'myservice' in the 'default' Kubernetes namespace.
func ByService(namespace, serviceName string) filters.Args {
	filter := filters.NewArgs()
	filter.Add("label", fmt.Sprintf("%s=%s", types.ServiceNameLabelKey, serviceName))
	filter.Add("label", fmt.Sprintf("%s=%s", types.NamespaceNameLabelKey, namespace))
	return filter
}

// AllPersistentVolumes creates a Docker filter argument that targets resources labeled with a Kubernetes persistent volume name.
// This function uses the types.PersistentVolumeNameLabelKey constant as the base label key to filter Docker resources.
//
// Parameters:
//   - None
//
// Returns:
// - filters.Args: A Docker filter object that can be used to filter Docker API calls based on the presence of the persistent volume name label.
//
// Usage Example:
//
//	filter := AllPersistentVolumes()
//	// Now 'filter' can be used in Docker API calls to filter resources that are labeled with any Kubernetes persistent volume name.
func AllPersistentVolumes() filters.Args {
	return filters.NewArgs(filters.Arg("label", types.PersistentVolumeNameLabelKey))
}
