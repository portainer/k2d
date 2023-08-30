package errors

import "errors"

// ErrResourceNotFound is an error returned when a Kubernetes resource is not found
var ErrResourceNotFound = errors.New("resource not found")
