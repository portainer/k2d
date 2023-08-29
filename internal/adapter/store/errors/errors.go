package errors

import "errors"

// ErrResourceNotFound is an error returned when a resource such as
// a ConfigMap or a Secret is not found.
var ErrResourceNotFound = errors.New("resource not found")
