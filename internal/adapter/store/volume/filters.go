package volume

import (
	"fmt"

	"github.com/docker/docker/api/types/filters"
	adapterfilters "github.com/portainer/k2d/internal/adapter/filters"
)

func configMapListFilter(namespace string) filters.Args {
	filter := adapterfilters.ByNamespace(namespace)
	filter.Add("label", fmt.Sprintf("%s=%s", ResourceTypeLabelKey, ConfigMapResourceType))
	return filter
}

func secretListFilter(namespace, secretKind string) filters.Args {
	filter := adapterfilters.ByNamespace(namespace)
	filter.Add("label", fmt.Sprintf("%s=%s", ResourceTypeLabelKey, secretKind))
	return filter
}
