package domain

import (
	"fmt"
	"sort"
)

// DeepMerge merges overlay on top of base. Maps are merged recursively.
// Lists and scalars are replaced by overlay. Null in overlay deletes the key.
// Type mismatches (e.g. map vs scalar) produce a MergeConflictError.
func DeepMerge(base, overlay map[string]any, path string, baseFile string, overlayFile string) (map[string]any, error) {
	result := make(map[string]any, len(base))
	for k, v := range base {
		result[k] = v
	}

	for k, overlayVal := range overlay {
		keyPath := joinPath(path, k)

		if overlayVal == nil {
			// Null in overlay = tombstone: delete the key
			delete(result, k)
			continue
		}

		baseVal, exists := result[k]
		if !exists || baseVal == nil {
			// Key doesn't exist in base or was nil: just set it
			result[k] = overlayVal
			continue
		}

		baseMap, baseIsMap := baseVal.(map[string]any)
		overlayMap, overlayIsMap := overlayVal.(map[string]any)

		if baseIsMap && overlayIsMap {
			merged, err := DeepMerge(baseMap, overlayMap, keyPath, baseFile, overlayFile)
			if err != nil {
				return nil, err
			}
			result[k] = merged
			continue
		}

		if baseIsMap != overlayIsMap {
			// One is a map, the other is not
			return nil, &MergeConflictError{
				Path:        keyPath,
				BaseType:    typeName(baseVal),
				OverlayType: typeName(overlayVal),
				BaseFile:    baseFile,
				OverlayFile: overlayFile,
			}
		}

		// Both are lists or both are scalars: replace
		result[k] = overlayVal
	}

	return result, nil
}

// MergeAll merges all discovered files in priority order.
// data maps file path → list of parsed documents.
func MergeAll(files []DiscoveredFile, data map[string][]map[string]any) (map[string]any, error) {
	result := map[string]any{}

	for _, file := range files {
		docs, ok := data[file.Path]
		if !ok {
			continue
		}

		for _, doc := range docs {
			merged, err := DeepMerge(result, doc, "", "", file.Path)
			if err != nil {
				return nil, fmt.Errorf("merging file %q: %w", file.RelPath, err)
			}
			result = merged
		}
	}

	return result, nil
}

// sortedKeys returns the keys of a map in sorted order.
func sortedKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func joinPath(parent, key string) string {
	if parent == "" {
		return key
	}
	return parent + "." + key
}

func typeName(v any) string {
	switch v.(type) {
	case map[string]any:
		return "map"
	case []any:
		return "list"
	case string:
		return "string"
	case bool:
		return "bool"
	case int, int64, float64:
		return "number"
	case nil:
		return "null"
	default:
		return fmt.Sprintf("%T", v)
	}
}
