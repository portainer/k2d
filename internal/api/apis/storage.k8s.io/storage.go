package storage

import (
	"github.com/emicklei/go-restful/v3"
	"github.com/portainer/k2d/internal/adapter"
	"github.com/portainer/k2d/internal/api/apis/storage.k8s.io/storageclasses"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type StorageService struct {
	storageclasses storageclasses.StorageClassService
}

func NewStorageService(adapter *adapter.KubeDockerAdapter) StorageService {
	return StorageService{
		storageclasses: storageclasses.NewStorageClassService(adapter),
	}
}

func (svc StorageService) GetAPIVersions(r *restful.Request, w *restful.Response) {
	apiVersion := metav1.APIVersions{
		TypeMeta: metav1.TypeMeta{
			Kind: "APIVersions",
		},
		Versions: []string{"storage.k8s.io/v1"},
	}

	w.WriteAsJson(apiVersion)
}

func (svc StorageService) ListAPIResources(r *restful.Request, w *restful.Response) {
	resourceList := metav1.APIResourceList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "APIResourceList",
			APIVersion: "v1",
		},
		GroupVersion: "storage.k8s.io/v1",
		APIResources: []metav1.APIResource{
			{
				Kind:         "StorageClass",
				SingularName: "",
				Name:         "storageclasses",
				ShortNames:   []string{"sc"},
				Verbs:        []string{"get,list"},
				Namespaced:   false,
			},
		},
	}

	w.WriteAsJson(resourceList)
}

func (svc StorageService) RegisterStorageAPI(routes *restful.WebService) {
	// storage
	svc.storageclasses.RegisterDeploymentAPI(routes)
}
