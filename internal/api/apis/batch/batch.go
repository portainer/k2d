package batch

import (
	"github.com/emicklei/go-restful/v3"
	"github.com/portainer/k2d/internal/adapter"
	"github.com/portainer/k2d/internal/api/apis/batch/jobs"
	"github.com/portainer/k2d/internal/controller"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type BatchService struct {
	jobs jobs.JobService
}

func NewBatchService(operations chan controller.Operation, adapter *adapter.KubeDockerAdapter) BatchService {
	return BatchService{
		jobs: jobs.NewJobService(adapter, operations),
	}
}

func (svc BatchService) GetAPIVersions(r *restful.Request, w *restful.Response) {
	apiVersion := metav1.APIVersions{
		TypeMeta: metav1.TypeMeta{
			Kind: "APIVersions",
		},
		Versions: []string{"batch/v1"},
	}

	w.WriteAsJson(apiVersion)
}

func (svc BatchService) ListAPIResources(r *restful.Request, w *restful.Response) {
	resourceList := metav1.APIResourceList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "APIResourceList",
			APIVersion: "v1",
		},
		GroupVersion: "batch/v1",
		APIResources: []metav1.APIResource{
			{
				Kind:         "Job",
				SingularName: "",
				Name:         "jobs",
				Verbs:        []string{"create", "list", "delete", "get", "patch"},
				Namespaced:   true,
			},
		},
	}

	w.WriteAsJson(resourceList)
}

func (svc BatchService) RegisterBatchAPI(routes *restful.WebService) {
	// jobs
	svc.jobs.RegisterJobAPI(routes)
}
