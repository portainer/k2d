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

// TODO: check if this is still useful and add comment
func RemoveItemsWithSuffix(items []string, suffix string) []string {
	var result []string
	for _, s := range items {
		if !strings.HasSuffix(s, suffix) {
			result = append(result, s)
		}
	}
	return result
}

// FilterStringsByPrefix takes a slice of strings and a prefix, and returns a new slice
// containing only the strings that start with the specified prefix.
func FilterStringsByPrefix(strs []string, prefix string) []string {
	var filtered []string
	for _, str := range strs {
		if strings.HasPrefix(str, prefix) {
			filtered = append(filtered, str)
		}
	}
	return filtered
}
