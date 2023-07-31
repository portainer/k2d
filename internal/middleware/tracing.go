package middleware

import (
	restful "github.com/emicklei/go-restful/v3"
	"github.com/portainer/k2d/internal/types"
	"k8s.io/apimachinery/pkg/util/uuid"
)

// AddTracingHeaders is a filter function that adds a unique tracing header to each incoming HTTP request.
// This tracing header ("X-K2d-Request-Id") is populated with a new UUID for each request.
// The function then proceeds with the rest of the filter chain by calling the ProcessFilter method.
func AddTracingHeaders(r *restful.Request, resp *restful.Response, chain *restful.FilterChain) {
	r.Request.Header.Set(types.RequestIDHeader, string(uuid.NewUUID()))
	chain.ProcessFilter(r, resp)
}
