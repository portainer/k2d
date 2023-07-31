package adapter

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubernetes/pkg/apis/core"
)

func (adapter *KubeDockerAdapter) ListNamespaces() core.NamespaceList {
	return core.NamespaceList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "NamespaceList",
			APIVersion: "v1",
		},
		Items: []core.Namespace{
			{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Namespace",
					APIVersion: "v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "default",
					CreationTimestamp: metav1.Time{
						Time: adapter.startTime,
					},
				},
				Status: core.NamespaceStatus{
					Phase: core.NamespaceActive,
				},
			},
		},
	}
}
