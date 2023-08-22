package apps

import (
	"github.com/emicklei/go-restful/v3"
	"github.com/portainer/k2d/internal/adapter"
	"github.com/portainer/k2d/internal/api/apis/apps/deployments"
	"github.com/portainer/k2d/internal/controller"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type AppsService struct {
	deployments deployments.DeploymentService
}

func NewAppsService(operations chan controller.Operation, adapter *adapter.KubeDockerAdapter) AppsService {
	return AppsService{
		deployments: deployments.NewDeploymentService(adapter, operations),
	}
}

func (svc AppsService) GetAPIVersions(r *restful.Request, w *restful.Response) {
	apiVersion := metav1.APIVersions{
		TypeMeta: metav1.TypeMeta{
			Kind: "APIVersions",
		},
		Versions: []string{"apps/v1"},
	}

	w.WriteAsJson(apiVersion)
}

func (svc AppsService) ListAPIResources(r *restful.Request, w *restful.Response) {
	resourceList := metav1.APIResourceList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "APIResourceList",
			APIVersion: "v1",
		},
		GroupVersion: "apps/v1",
		APIResources: []metav1.APIResource{
			{
				Kind:         "Deployment",
				SingularName: "",
				Name:         "deployments",
				Verbs:        []string{"create", "list", "delete", "get", "patch"},
				Namespaced:   true,
			},
		},
	}

	w.WriteAsJson(resourceList)
}

func (svc AppsService) RegisterAppsAPI(routes *restful.WebService) {
	// deployments
	svc.deployments.RegisterDeploymentAPI(routes)
}
