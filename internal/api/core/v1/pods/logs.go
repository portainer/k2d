package pods

import (
	"fmt"
	"io"
	"net/http"

	"github.com/emicklei/go-restful/v3"
	"github.com/portainer/k2d/internal/adapter"
	"github.com/portainer/k2d/internal/api/utils"
)

func (svc PodService) GetPodLogs(r *restful.Request, w *restful.Response) {
	podName := r.PathParameter("name")

	podLogOptions := adapter.PodLogOptions{
		Follow:     r.QueryParameter("follow") == "true",
		Timestamps: r.QueryParameter("timestamps") == "true",
		Tail:       r.QueryParameter("tailLines"),
	}

	logs, err := svc.adapter.GetPodLogs(r.Request.Context(), podName, podLogOptions)
	if err != nil {
		utils.HttpError(r, w, http.StatusInternalServerError, fmt.Errorf("unable to get pod logs: %w", err))
		return
	}
	defer logs.Close()

	if !podLogOptions.Follow {
		_, err = io.Copy(w, logs)
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
		buffer := make([]byte, 1024)
		for {
			size, err := logs.Read(buffer)
			if err != nil {
				if err != io.EOF {
					w.WriteError(http.StatusInternalServerError, err)
				}
				break
			}

			_, err = w.Write(buffer[:size])
			if err != nil {
				break
			}

			flusher.Flush()
		}
	} else {
		// Fallback to normal HTTP response
		_, _ = io.Copy(w, logs)
	}
}
