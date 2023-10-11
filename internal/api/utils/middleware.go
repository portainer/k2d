package utils

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/emicklei/go-restful/v3"
	"github.com/portainer/k2d/internal/adapter"
	adaptererr "github.com/portainer/k2d/internal/adapter/errors"
	"github.com/portainer/k2d/internal/logging"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// NamespaceValidation is a filter function that validates the existence of a namespace in the k2d environment.
// The function attempts to retrieve the namespace specified in the request path and checks its existence.
// If the namespace is not found, it responds with an HTTP 404 status and an appropriate error message.
// If an internal server error occurs, it responds with an HTTP 500 status.
// If the namespace is valid, the function adds it as an attribute to the request object and continues
// the request processing by invoking the next filter in the chain.
//
// Note: This filter is not used by any DELETE endpoints because they are usually called sequentially by
// Kubernetes clients. The namespace is usually deleted before the validation can be performed.
//
// Parameters:
//   - adapter: A pointer to an initialized KubeDockerAdapter object.
//
// Returns:
//   - restful.FilterFunction: A function conforming to the FilterFunction type from the go-restful package.
func NamespaceValidation(adapter *adapter.KubeDockerAdapter) restful.FilterFunction {
	return func(r *restful.Request, w *restful.Response, chain *restful.FilterChain) {
		namespace := r.PathParameter("namespace")

		_, err := adapter.GetNamespace(r.Request.Context(), namespace)
		if err != nil {
			if errors.Is(err, adaptererr.ErrResourceNotFound) {
				notFoundErr := apierr.NewNotFound(
					schema.GroupResource{Group: "", Resource: "namespaces"},
					namespace,
				)

				notFoundErr.ErrStatus.TypeMeta = metav1.TypeMeta{
					Kind:       "Status",
					APIVersion: "v1",
				}

				logger := logging.LoggerFromContext(r.Request.Context())
				logger.Errorw("namespace not found", "namespace", namespace)

				w.WriteHeaderAndEntity(http.StatusNotFound, notFoundErr.ErrStatus)
				return
			}

			HttpError(r, w, http.StatusInternalServerError, fmt.Errorf("unable to get namespace: %w", err))
			return
		}

		r.SetAttribute("namespace", namespace)
		chain.ProcessFilter(r, w)
	}
}
