package utils

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/emicklei/go-restful/v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// UnsupportedOperation is a helper function that writes a 404 Not Found response to the HTTP response.
func UnsupportedOperation(r *restful.Request, w *restful.Response) {
	w.WriteHeader(http.StatusNotFound)
}

// listFunc defines a function type that receives a context and returns a list of objects.
type listFunc func(ctx context.Context) (interface{}, error)

// getTableFunc defines a function type that receives a context and returns a metav1.Table reference.
type getTableFunc func(ctx context.Context) (*metav1.Table, error)

// ListResources handles the HTTP request for listing resources. It supports two response modes,
// standard list response, and table response. The mode is determined by the "Accept" HTTP header.
// If the header value is "application/json;as=Table;v=v1;g=meta.k8s.io", a table response is returned.
// Otherwise, a standard list response is returned. It uses provided listFunc and getTableFunc
// to fetch the data. It will handle any errors that occur during data retrieval and write them
// to the HTTP response as necessary. Successful data retrieval results in the data being written
// to the HTTP response in JSON format.
//
// Parameters:
// r: The incoming RESTful request containing information such as the context and HTTP headers.
// w: The RESTful response writer to write the HTTP response.
// listFunc: A function that fetches a list of resources.
// getTableFunc: A function that fetches a table of resources.
func ListResources(r *restful.Request, w *restful.Response, listFunc listFunc, getTableFunc getTableFunc) {
	acceptHeader := r.Request.Header.Get("Accept")

	if strings.Contains(acceptHeader, "application/json;as=Table;v=v1;g=meta.k8s.io") {
		table, err := getTableFunc(r.Request.Context())
		if err != nil {
			HttpError(r, w, http.StatusInternalServerError, fmt.Errorf("unable to get table: %w", err))
			return
		}

		w.WriteAsJson(table)
		return
	}

	list, err := listFunc(r.Request.Context())
	if err != nil {
		HttpError(r, w, http.StatusInternalServerError, fmt.Errorf("unable to list resources: %w", err))
		return
	}

	w.WriteAsJson(list)
}
