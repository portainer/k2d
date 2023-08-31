package strings

import (
	"strings"
)

// RetrieveUniquePrefixes returns a slice containing unique prefixes extracted from a given slice of strings.
// Each string in the original slice is split by a specified separator, and the prefix (i.e., the part
// of the string before the separator) is stored.
//
// The function performs the following steps:
// 1. Creates an empty map named 'uniqueMap' to store unique prefixes.
// 2. Iterates over each string 'str' in the provided 'data' slice.
// 3. Splits 'str' into parts separated by the 'separator' string, and stores the prefix part in 'uniqueMap'.
// 4. Initializes an empty slice named 'uniqueSlice' to hold the unique prefixes.
// 5. Iterates through the keys of 'uniqueMap' and appends each key to 'uniqueSlice'.
//
// Parameters:
// - data: The slice of strings to extract unique prefixes from.
// - separator: The string that separates the prefix from the rest of each string in 'data'.
//
// Returns:
// - A slice containing unique prefixes from the original 'data' slice.
//
// Example:
//
//	inputData := []string{"apple:fruit", "banana:fruit", "apple:tech", "car:vehicle"}
//	separator := ":"
//	outputData := RetrieveUniquePrefixes(inputData, separator)
//	// outputData will be ["apple", "banana", "car"]
func RetrieveUniquePrefixes(data []string, separator string) []string {
	uniqueMap := make(map[string]struct{})
	for _, str := range data {
		prefix := strings.Split(str, separator)[0]
		uniqueMap[prefix] = struct{}{}
	}

	var uniqueSlice []string
	for key := range uniqueMap {
		uniqueSlice = append(uniqueSlice, key)
	}

	return uniqueSlice
}

// RemoveItemsWithSuffix filters out elements in a string slice that have a specific suffix.
//
// The function performs the following steps:
// 1. Initializes an empty string slice called 'result' to store the filtered items.
// 2. Iterates through each string 's' in the given 'items' slice.
// 3. Checks if the string 's' has the specified suffix using the strings.HasSuffix function.
// 4. If the string does not have the suffix, appends it to the 'result' slice.
//
// Parameters:
// - items: The slice of strings to be filtered.
// - suffix: The suffix string to check against each element in the 'items' slice.
//
// Returns:
// - A new slice containing all strings from 'items' that do not end with the specified 'suffix'.
//
// Example:
//
//	inputItems := []string{"apple.txt", "banana", "cherry.log", "date"}
//	suffix := ".txt"
//	outputItems := RemoveItemsWithSuffix(inputItems, suffix)
//	// outputItems will be ["banana", "cherry.log", "date"]
func RemoveItemsWithSuffix(items []string, suffix string) []string {
	var result []string
	for _, s := range items {
		if !strings.HasSuffix(s, suffix) {
			result = append(result, s)
		}
	}
	return result
}

// FilterStringsByPrefix returns a new slice containing only the strings from the original slice
// that start with a specific prefix.
//
// The function performs the following steps:
// 1. Initializes an empty string slice called 'filtered' to store the filtered items.
// 2. Iterates through each string 'str' in the given 'strs' slice.
// 3. Checks if the string 'str' starts with the specified prefix using the strings.HasPrefix function.
// 4. If the string starts with the prefix, it is appended to the 'filtered' slice.
//
// Parameters:
// - strs: The slice of strings to be filtered.
// - prefix: The prefix string to check against each element in the 'strs' slice.
//
// Returns:
// - A new slice containing all strings from 'strs' that start with the specified 'prefix'.
//
// Example:
//
//	inputStrings := []string{"apple", "banana", "apricot", "cherry"}
//	prefix := "ap"
//	outputStrings := FilterStringsByPrefix(inputStrings, prefix)
//	// outputStrings will be ["apple", "apricot"]
func FilterStringsByPrefix(strs []string, prefix string) []string {
	var filtered []string
	for _, str := range strs {
		if strings.HasPrefix(str, prefix) {
			filtered = append(filtered, str)
		}
	}
	return filtered
}
