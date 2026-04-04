package domain

// StripReservedKeys recursively removes _templates and _inherit keys from the tree.
func StripReservedKeys(tree map[string]any, templatesKey, inheritKey string) map[string]any {
	result := make(map[string]any, len(tree))
	for k, v := range tree {
		if k == templatesKey || k == inheritKey {
			continue
		}
		if child, ok := v.(map[string]any); ok {
			result[k] = StripReservedKeys(child, templatesKey, inheritKey)
		} else {
			result[k] = v
		}
	}
	return result
}
