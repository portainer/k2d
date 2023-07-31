package authorization

import (
	"fmt"
	"net/http"

	"github.com/emicklei/go-restful/v3"
	"github.com/portainer/k2d/internal/api/utils"
	httputils "github.com/portainer/k2d/pkg/http"
	v1 "k8s.io/api/authorization/v1"
)

func (svc AuthorizationService) CreateSelfSubjectAccessReview(r *restful.Request, w *restful.Response) {
	accessReview := &v1.SelfSubjectAccessReview{}

	err := httputils.ParseJSONBody(r.Request, &accessReview)
	if err != nil {
		utils.HttpError(r, w, http.StatusBadRequest, fmt.Errorf("unable to parse request body: %w", err))
		return
	}

	// We simply update the status of the access review to allow everything
	accessReview.Status = v1.SubjectAccessReviewStatus{
		Allowed: true,
		Denied:  false,
		Reason:  "k2d does not implement authorization",
	}

	w.WriteAsJson(accessReview)
}
