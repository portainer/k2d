package errors

import "errors"

// TODO: replace usage with adapter/errors
// ErrResourceNotFound is an error returned when a resource such as
// a ConfigMap or a Secret is not found.
var ErrResourceNotFound = errors.New("resource not found")
