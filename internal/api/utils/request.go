package utils

import "github.com/emicklei/go-restful/v3"

// NamespaceParam returns the namespace parameter from the request.
// If the namespace is empty, it returns "default".
func NamespaceParameter(r *restful.Request) string {
	namespace := r.PathParameter("namespace")

	if namespace == "" {
		namespace = "default"
	}

	return namespace
}
