package maputils

// ContainsKeyValuePairInMap returns true if the key/value pair exists in the map
func ContainsKeyValuePairInMap(key string, value string, m map[string]string) bool {
	if val, ok := m[key]; ok {
		return val == value
	}
	return false
}

// mergeMapsInPlace takes two maps of type map[string]string.
// It copies the key-value pairs from the second map (map2) into the first map (map1),
// overwriting any existing values with the same keys.
// This operation is performed in place, meaning that the first map (map1) will be modified directly,
// and the merged result will be stored in map1.
func MergeMapsInPlace(map1, map2 map[string]string) {
	for key, value := range map2 {
		map1[key] = value
	}
}

// ConvertMapStringToStringSliceByte takes a map with string keys and string values,
// and returns a new map with the same keys but with the values converted to byte slices.
//
// Parameters:
// - input: A map[string]string that you want to convert.
//
// Returns:
// A new map[string][]byte where the keys are the same as in the input map, and the
// values are byte slices converted from the corresponding string values in the input map.
func ConvertMapStringToStringSliceByte(input map[string]string) map[string][]byte {
	output := make(map[string][]byte, len(input))
	for k, v := range input {
		output[k] = []byte(v)
	}
	return output
}
