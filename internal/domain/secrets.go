package domain

import (
	"fmt"
	"strings"
)

const sentinelPrefix = "__CONFSTACK_SECRET_"
const sentinelSuffix = "__"

// ResolveSecrets walks the merged tree and replaces sentinel strings with their real values.
// It produces two copies: redacted (sentinels replaced with "(sensitive)") and full (real values).
// It also records which dot-paths contain secrets.
func ResolveSecrets(tree map[string]any, sentinelMap map[string]string) (redacted, full map[string]any, secretPaths map[string]bool, err error) {
	secretPaths = make(map[string]bool)
	redacted, err = resolveMap(tree, sentinelMap, secretPaths, "", true)
	if err != nil {
		return nil, nil, nil, err
	}
	full, err = resolveMap(tree, sentinelMap, nil, "", false)
	if err != nil {
		return nil, nil, nil, err
	}
	return redacted, full, secretPaths, nil
}

func resolveMap(m map[string]any, sentinelMap map[string]string, secretPaths map[string]bool, pathPrefix string, redact bool) (map[string]any, error) {
	result := make(map[string]any, len(m))
	for k, v := range m {
		path := joinPath(pathPrefix, k)
		resolved, err := resolveValue(v, sentinelMap, secretPaths, path, redact)
		if err != nil {
			return nil, err
		}
		result[k] = resolved
	}
	return result, nil
}

func resolveValue(v any, sentinelMap map[string]string, secretPaths map[string]bool, path string, redact bool) (any, error) {
	switch val := v.(type) {
	case map[string]any:
		return resolveMap(val, sentinelMap, secretPaths, path, redact)
	case []any:
		result := make([]any, len(val))
		for i, item := range val {
			itemPath := fmt.Sprintf("%s[%d]", path, i)
			resolved, err := resolveValue(item, sentinelMap, secretPaths, itemPath, redact)
			if err != nil {
				return nil, err
			}
			result[i] = resolved
		}
		return result, nil
	case string:
		return resolveString(val, sentinelMap, secretPaths, path, redact), nil
	default:
		return v, nil
	}
}

func resolveString(s string, sentinelMap map[string]string, secretPaths map[string]bool, path string, redact bool) any {
	if !isSentinel(s) {
		return s
	}
	if secretPaths != nil {
		secretPaths[path] = true
	}
	if redact {
		return "(sensitive)"
	}
	if real, ok := sentinelMap[s]; ok {
		return real
	}
	// Sentinel not found in map (shouldn't happen): return as-is
	return s
}

func isSentinel(s string) bool {
	return strings.HasPrefix(s, sentinelPrefix) && strings.HasSuffix(s, sentinelSuffix)
}
