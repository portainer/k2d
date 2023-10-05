package utils

import (
	"github.com/emicklei/go-restful/v3"
)

// GetNamespaceFromRequest attempts to obtain the namespace from a restful.Request object.
// Initially, it looks for an attribute named "namespace" that may have been set by preceding middleware (e.g., NamespaceValidation).
// If this attribute is either not found or empty, the function falls back to retrieving the namespace from the request's path parameters.
//
// Parameters:
//   - r: A pointer to a restful.Request object containing either an attribute or a path parameter named "namespace".
//
// Returns:
//   - string: The namespace retrieved from the request attributes or path parameters. Returns an empty string if not found.
func GetNamespaceFromRequest(r *restful.Request) string {
	namespace, ok := r.Attribute("namespace").(string)

	if !ok || namespace == "" {
		namespace = r.PathParameter("namespace")
	}

	return namespace
}
