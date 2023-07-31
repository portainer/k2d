package middleware

import (
	restful "github.com/emicklei/go-restful/v3"
	"github.com/portainer/k2d/internal/logging"
	"github.com/portainer/k2d/internal/types"
)

// LogRequests is a filter function that logs the details of each incoming HTTP request.
// The function extracts a logger from the request's context and logs key details such as the request URL,
// HTTP method, remote address, and a unique request ID header ("X-K2d-Request-Id").
// After logging, the function continues processing the rest of the filter chain by calling the ProcessFilter method.
func LogRequests(r *restful.Request, resp *restful.Response, chain *restful.FilterChain) {
	logger := logging.LoggerFromContext(r.Request.Context())

	logger.Debugw("received HTTP request",
		"url", r.Request.URL,
		"method", r.Request.Method,
		"remote_address", r.Request.RemoteAddr,
		"request_id", r.Request.Header.Get(types.RequestIDHeader),
		"header_accept", r.Request.Header.Get("Accept"),
	)

	chain.ProcessFilter(r, resp)
}
