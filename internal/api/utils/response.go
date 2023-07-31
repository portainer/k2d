package utils

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/emicklei/go-restful/v3"
	"github.com/portainer/k2d/internal/k8s"
	"k8s.io/apimachinery/pkg/runtime"
)

type convertFunc func(src, dest interface{}) error

// WriteListBasedOnAcceptHeader processes the list of Kubernetes objects, converts it to the appropriate format,
// and writes it to the HTTP response. The format of the output (table or JSON) is determined
// by the Accept header of the incoming request. The function accepts a list of Kubernetes objects,
// a function to create an empty list of the appropriate runtime.Object type, and a function to convert
// the Kubernetes objects to the appropriate versioned type.
//
// The steps of the function are as follows:
//  1. Check the Accept header in the request.
//  2. If the Accept header requests a Table (application/json;as=Table;v=v1;g=meta.k8s.io), convert the
//     list to a table:
//     a. Assert that the list is a runtime.Object.
//     b. Generate a table from the list.
//     c. Write the table as JSON to the HTTP response.
//  3. If the Accept header does not request a Table, convert the list to a versioned type and write it as
//     JSON to the HTTP response:
//     a. Create an empty list of the appropriate versioned type.
//     b. Convert the list to the versioned type using the provided conversion function.
//     c. Write the converted list as JSON to the HTTP response.
//
// Parameters:
// - request: the incoming HTTP request.
// - response: the HTTP response writer.
// - k8sObjects: the list of Kubernetes objects to process.
// - createNewEmptyList: a function that returns a new, empty list of the appropriate runtime.Object type.
// - conversionFunction: a function that converts the list of Kubernetes objects to the appropriate versioned type.
func WriteListBasedOnAcceptHeader(request *restful.Request, response *restful.Response, k8sObjects interface{}, createNewEmptyList func() runtime.Object, conversionFunction convertFunc) {

	// Check the Accept header in the request
	acceptHeader := request.Request.Header.Get("Accept")

	// If the Accept header requests a Table, convert the list to a table
	if strings.Contains(acceptHeader, "application/json;as=Table;v=v1;g=meta.k8s.io") {
		runtimeObjectList, isRuntimeObject := k8sObjects.(runtime.Object)
		if !isRuntimeObject {
			HttpError(request, response, http.StatusInternalServerError, fmt.Errorf("unable to convert list to runtime.Object"))
			return
		}

		// Generate the table
		table, err := k8s.GenerateTable(runtimeObjectList)
		if err != nil {
			HttpError(request, response, http.StatusInternalServerError, fmt.Errorf("unable to generate table: %w", err))
			return
		}

		// Write the table as JSON to the response
		response.WriteAsJson(table)
		return
	}

	// If the Accept header does not request a Table, convert the list to a runtime.Object and write it as JSON
	// First, create an empty list to store the converted objects
	convertedObjectList := createNewEmptyList()

	// Convert the list
	err := conversionFunction(k8sObjects, convertedObjectList)
	if err != nil {
		HttpError(request, response, http.StatusInternalServerError, fmt.Errorf("unable to convert list: %w", err))
		return
	}

	// Write the converted list as JSON to the response
	response.WriteAsJson(convertedObjectList)
}

// UnsupportedOperation is a helper function that writes a 404 Not Found response to the HTTP response.
func UnsupportedOperation(r *restful.Request, w *restful.Response) {
	w.WriteHeader(http.StatusNotFound)
}
