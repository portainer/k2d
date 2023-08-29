package pods

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/docker/docker/pkg/stdcopy"
	"github.com/emicklei/go-restful/v3"
	"github.com/portainer/k2d/internal/adapter"
	"github.com/portainer/k2d/internal/api/utils"
)

// GetPodLogs handles the HTTP request for retrieving logs from a pod.
// It fetches the logs using the provided adapter and writes them to the HTTP response.
// If the "follow" query parameter is set to true, it streams the logs to the response.
// This function sets the necessary headers for streaming and uses a custom writer
// (flushWriter) that invokes the http.Flusher interface on every Write call to ensure
// the data is immediately sent to the client.
func (svc PodService) GetPodLogs(r *restful.Request, w *restful.Response) {
	podName := r.PathParameter("name")
	namespaceName := r.PathParameter("namespace")

	podLogOptions := adapter.PodLogOptions{
		Follow:     r.QueryParameter("follow") == "true",
		Timestamps: r.QueryParameter("timestamps") == "true",
		Tail:       r.QueryParameter("tailLines"),
	}

	logs, err := svc.adapter.GetPodLogs(context.Background(), namespaceName, podName, podLogOptions)
	if err != nil {
		utils.HttpError(r, w, http.StatusInternalServerError, fmt.Errorf("unable to get pod logs: %w", err))
		return
	}
	defer logs.Close()

	if !podLogOptions.Follow {
		_, err := stdcopy.StdCopy(w, w, logs)
		if err != nil {
			utils.HttpError(r, w, http.StatusInternalServerError, fmt.Errorf("unable to copy logs: %w", err))
			return
		}
		return
	}

	// Set headers related to event streaming
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	// Use a flusher to allow streaming data in the HTTP response
	if flusher, ok := w.ResponseWriter.(http.Flusher); ok {
		fw := &flushWriter{w: w, flusher: flusher}
		_, err = stdcopy.StdCopy(fw, w, logs)
		if err != nil {
			utils.HttpError(r, w, http.StatusInternalServerError, fmt.Errorf("unable to stream logs: %w", err))
			return
		}
	} else {
		// Fallback to normal HTTP response
		_, _ = stdcopy.StdCopy(w, w, logs)
	}
}

// flushWriter is a custom io.Writer that wraps another io.Writer
// (in this case, an http.ResponseWriter), and a http.Flusher.
// Every time Write is called, it also calls Flush on the Flush interface,
// to immediately send the data to the client.
type flushWriter struct {
	w       io.Writer
	flusher http.Flusher
}

// Write writes the given bytes to the underlying writer and then calls Flush.
// It returns the number of bytes written and any write error encountered.
func (fw *flushWriter) Write(p []byte) (n int, err error) {
	n, err = fw.w.Write(p)
	if err != nil {
		return n, err
	}
	fw.flusher.Flush()
	return n, nil
}
