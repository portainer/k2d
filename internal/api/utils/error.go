package utils

import (
	"github.com/emicklei/go-restful/v3"
	"github.com/portainer/k2d/internal/logging"
	"github.com/portainer/k2d/internal/types"
	"go.uber.org/zap"
)

// HttpError logs an error and sends an HTTP response with the given status code and error message.
// The function retrieves the logger from the context of the HTTP request, logs the error,
// and then writes the error message to the HTTP response with the specified status code.
//
// Parameters:
// - r: A pointer to the incoming restful.Request from which the logger is retrieved.
// - w: A pointer to the restful.Response where the error message will be written.
// - statusCode: The HTTP status code to be sent in the response.
// - err: The error to be logged and sent in the response.
func HttpError(r *restful.Request, w *restful.Response, statusCode int, err error) {
	logging.LoggerFromContext(r.Request.Context()).
		WithOptions(zap.AddCallerSkip(1)).
		With(zap.String("request_id", r.Request.Header.Get(types.RequestIDHeader))).
		Error(err)

	w.WriteError(statusCode, err)
}
