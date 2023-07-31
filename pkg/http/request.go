package http

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// ParseJSONBody is a helper function that reads the entire body from an HTTP request
// and then attempts to decode the JSON content into the provided 'data' interface.
// It first reads the body of the HTTP request into a byte slice.
// If this is successful, it tries to unmarshal the JSON from the byte slice into the 'data' interface.
// If the JSON unmarshalling is successful, the function will return nil, otherwise, it will return an error.
// If any step fails, an error is returned, and it will be wrapped with additional context for easier debugging.
func ParseJSONBody(req *http.Request, data interface{}) error {
	body, err := io.ReadAll(req.Body)
	if err != nil {
		return fmt.Errorf("unable to read request body: %w", err)
	}

	err = json.Unmarshal(body, data)
	if err != nil {
		return fmt.Errorf("unable to parse JSON from request body: %w", err)
	}

	return nil
}
