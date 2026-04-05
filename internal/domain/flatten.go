package domain

// Flatten recursively walks a nested map and produces separator-delimited keys.
// Only maps are recursed into; leaf values (strings, numbers, bools, lists, nil) are kept as-is.
func Flatten(data map[string]any, separator string) map[string]any {
	result := make(map[string]any)
	flattenRecursive(data, "", separator, result)
	return result
}

func flattenRecursive(data map[string]any, prefix, separator string, result map[string]any) {
	for k, v := range data {
		fullKey := k
		if prefix != "" {
			fullKey = prefix + separator + k
		}
		if child, ok := v.(map[string]any); ok {
			flattenRecursive(child, fullKey, separator, result)
		} else {
			result[fullKey] = v
		}
	}
}
