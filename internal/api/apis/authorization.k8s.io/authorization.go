package authorization

import (
	"github.com/emicklei/go-restful/v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type AuthorizationService struct {
}

func NewAuthorizationService() AuthorizationService {
	return AuthorizationService{}
}

func (svc AuthorizationService) GetAPIVersions(r *restful.Request, w *restful.Response) {
	apiVersion := metav1.APIVersions{
		TypeMeta: metav1.TypeMeta{
			Kind: "APIVersions",
		},
		Versions: []string{"authorization.k8s.io/v1"},
	}

	w.WriteAsJson(apiVersion)
}

func (svc AuthorizationService) ListAPIResources(r *restful.Request, w *restful.Response) {
	resourceList := metav1.APIResourceList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "APIResourceList",
			APIVersion: "v1",
		},
		GroupVersion: "authorization.k8s.io/v1",
		APIResources: []metav1.APIResource{
			{
				Kind:         "SelfSubjectAccessReview",
				SingularName: "",
				Name:         "selfsubjectaccessreviews",
				Verbs:        []string{"create"},
				Namespaced:   false,
			},
		},
	}

	w.WriteAsJson(resourceList)
}

func (svc AuthorizationService) RegisterAuthorizationAPI(routes *restful.WebService) {
	// selfsubjectaccessreviews
	routes.Route(routes.POST("/v1/selfsubjectaccessreviews").
		To(svc.CreateSelfSubjectAccessReview))
}
