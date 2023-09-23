package apis

import (
	"github.com/emicklei/go-restful/v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func ListAPIGroups(r *restful.Request, w *restful.Response) {
	groupList := metav1.APIGroupList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "APIGroupList",
			APIVersion: "v1",
		},
		Groups: []metav1.APIGroup{
			{
				Name: "apps",
				Versions: []metav1.GroupVersionForDiscovery{
					{
						GroupVersion: "apps/v1",
						Version:      "v1",
					},
				},
			},
			{
				Name: "events.k8s.io",
				Versions: []metav1.GroupVersionForDiscovery{
					{
						GroupVersion: "events.k8s.io/v1",
						Version:      "v1",
					},
				},
			},
			{
				Name: "authorization.k8s.io",
				Versions: []metav1.GroupVersionForDiscovery{
					{
						GroupVersion: "authorization.k8s.io/v1",
						Version:      "v1",
					},
				},
			},
			{
				Name: "storage.k8s.io",
				Versions: []metav1.GroupVersionForDiscovery{
					{
						GroupVersion: "storage.k8s.io/v1",
						Version:      "v1",
					},
				},
			},
			{
				Name: "metrics.k8s.io",
				Versions: []metav1.GroupVersionForDiscovery{
					{
						GroupVersion: "metrics.k8s.io/v1beta1",
						Version:      "v1beta1",
					},
				},
			},
		},
	}

	w.WriteAsJson(groupList)
}
