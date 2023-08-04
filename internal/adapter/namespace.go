package adapter

import (
	"fmt"

	"github.com/portainer/k2d/internal/k8s"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubernetes/pkg/apis/core"
)

func (adapter *KubeDockerAdapter) ListNamespaces() (corev1.NamespaceList, error) {
	namespaceList := adapter.listNamespaces()

	versionedNamespaceList := corev1.NamespaceList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "NamespaceList",
			APIVersion: "v1",
		},
	}

	err := adapter.ConvertK8SResource(&namespaceList, &versionedNamespaceList)
	if err != nil {
		return corev1.NamespaceList{}, fmt.Errorf("unable to convert internal NamespaceList to versioned NamespaceList: %w", err)
	}

	return versionedNamespaceList, nil
}

func (adapter *KubeDockerAdapter) GetNamespaceTable() (*metav1.Table, error) {
	namespaceList := adapter.listNamespaces()
	return k8s.GenerateTable(&namespaceList)
}

func (adapter *KubeDockerAdapter) listNamespaces() core.NamespaceList {
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
