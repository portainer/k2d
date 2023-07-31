package k8s

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/kubernetes/pkg/printers"
	printersinternal "k8s.io/kubernetes/pkg/printers/internalversion"
)

// GenerateTable generates a table representation of the provided runtime object.
// The function returns a pointer to the generated table (*metav1.Table) and an error if any.
func GenerateTable(obj runtime.Object) (*metav1.Table, error) {
	tableGenerator := printers.NewTableGenerator()
	printersinternal.AddHandlers(tableGenerator)

	table, err := tableGenerator.GenerateTable(obj, printers.GenerateOptions{})
	if err != nil {
		return nil, err
	}

	table.TypeMeta = metav1.TypeMeta{
		Kind:       "Table",
		APIVersion: "meta.k8s.io/v1",
	}

	return table, nil
}
