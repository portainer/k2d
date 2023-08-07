package strings

import (
	"strings"
)

// UniquePrefixes returns a slice of unique prefixes from a slice of strings
// Prefixes are retrieved based on the specified separator
func UniquePrefixes(data []string, separator string) []string {
	uniqueMap := make(map[string]struct{})
	for _, str := range data {
		prefix := strings.Split(str, separator)[0]
		uniqueMap[prefix] = struct{}{}
	}

	// Convert map keys to slice
	var uniqueSlice []string
	for key := range uniqueMap {
		uniqueSlice = append(uniqueSlice, key)
	}

	return uniqueSlice
}

// IsStringInSlice checks if the target string is present in the provided slice of strings.
// It returns true if the target string is found in the slice, and false otherwise.
func IsStringInSlice(target string, slice []string) bool {
	for _, item := range slice {
		if item == target {
			return true
		}
	}
	return false
}

func RemoveItemsWithSuffix(items []string, suffix string) []string {
	var result []string
	for _, s := range items {
		if !strings.HasSuffix(s, suffix) {
			result = append(result, s)
		}
	}
	return result
}
