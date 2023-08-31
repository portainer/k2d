package k8s

import (
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/kubernetes/pkg/printers"
	printersinternal "k8s.io/kubernetes/pkg/printers/internalversion"
)

// GenerateTable takes a Kubernetes runtime.Object, expected to be a kind of resource list
// (e.g., PodList, ServiceList), and converts it into a metav1.Table format. This table format
// is compatible with the tabular output format used by kubectl for human-readable views.
//
// Detailed Steps:
//  1. Utilize the Kubernetes internal print handlers to generate a basic table from the runtime.Object input.
//     The print handlers populate the table's column definitions and rows based on the resource type.
//
// 2. Use meta.ExtractList to convert the runtime.Object into a slice of individual resources.
//
//   - meta.ExtractList extracts elements from a list object and returns them as a slice.
//
//   - This is essential to iterate over the list and manipulate each resource individually.
//
//     3. Iterate over each element in the slice and create a PartialObjectMetadata object from it.
//     PartialObjectMetadata contains only the resource's metadata, significantly reducing the data size.
//
// 4. Replace the original Object field in each row of the table with this PartialObjectMetadata.
//
//  5. Finally, assign the appropriate TypeMeta to the table to specify it as a metav1.Table of API version
//     "meta.k8s.io/v1".
//
// The function returns:
// - A pointer to the generated metav1.Table if successful.
// - An error if the input is not a list or if table generation fails for some reason.
//
// Note: This function is intended for use with resource list types such as PodList, ServiceList, etc.
//
// Parameters:
// - obj: A Kubernetes runtime.Object, expected to be a kind of resource list.
//
// Returns:
// - A pointer to a metav1.Table.
// - An error if table generation fails.
func GenerateTable(obj runtime.Object) (*metav1.Table, error) {
	tableGenerator := printers.NewTableGenerator()
	printersinternal.AddHandlers(tableGenerator)

	options := printers.GenerateOptions{}
	table, err := tableGenerator.GenerateTable(obj, options)
	if err != nil {
		return nil, err
	}

	// Convert the runtime object to a list with meta accessor
	if list, err := meta.ExtractList(obj); err == nil {
		for i := range table.Rows {
			if i >= len(list) {
				break
			}
			runtimeObj := list[i]
			gvk := runtimeObj.GetObjectKind().GroupVersionKind()

			if metaObj, ok := runtimeObj.(metav1.Object); ok {
				partialMetadata := &metav1.PartialObjectMetadata{
					TypeMeta: metav1.TypeMeta{
						Kind:       gvk.Kind,
						APIVersion: gvk.GroupVersion().String(),
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      metaObj.GetName(),
						Namespace: metaObj.GetNamespace(),
					},
				}
				table.Rows[i].Object.Object = partialMetadata
			}
		}
	}

	table.TypeMeta = metav1.TypeMeta{
		Kind:       "Table",
		APIVersion: "meta.k8s.io/v1",
	}

	return table, nil
}
