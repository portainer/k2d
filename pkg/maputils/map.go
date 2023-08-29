package maputils

// ContainsKeyValuePairInMap returns true if the key/value pair exists in the map
func ContainsKeyValuePairInMap(key string, value string, m map[string]string) bool {
	if val, ok := m[key]; ok {
		return val == value
	}
	return false
}
