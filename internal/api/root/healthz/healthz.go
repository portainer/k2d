package healthz

import (
	"net/http"

	"github.com/emicklei/go-restful/v3"
)

type HealthzService struct{}

func NewHealthzService() HealthzService {
	return HealthzService{}
}

func (svc HealthzService) Healthz(r *restful.Request, w *restful.Response) {
	w.WriteHeader(http.StatusOK)
}
