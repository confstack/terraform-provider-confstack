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
	sentinels := findSentinels(s)
	if len(sentinels) == 0 {
		return s
	}

	if secretPaths != nil {
		secretPaths[path] = true
	}

	// Exact sentinel: the whole string is a single sentinel — preserve non-string return type
	if len(sentinels) == 1 && s == sentinels[0] {
		if redact {
			return "(sensitive)"
		}
		if real, ok := sentinelMap[s]; ok {
			return real
		}
		// Sentinel not found in map (shouldn't happen): return as-is
		return s
	}

	// Inline sentinel(s): replace each occurrence within the larger string
	if redact {
		result := s
		for _, sentinel := range sentinels {
			result = strings.ReplaceAll(result, sentinel, "(sensitive)")
		}
		return result
	}
	result := s
	for _, sentinel := range sentinels {
		if real, ok := sentinelMap[sentinel]; ok {
			result = strings.ReplaceAll(result, sentinel, real)
		}
	}
	return result
}

// findSentinels returns all sentinel substrings found in s.
func findSentinels(s string) []string {
	var sentinels []string
	for i := 0; i < len(s); {
		start := strings.Index(s[i:], sentinelPrefix)
		if start == -1 {
			break
		}
		start += i
		end := strings.Index(s[start+len(sentinelPrefix):], sentinelSuffix)
		if end == -1 {
			break
		}
		end = start + len(sentinelPrefix) + end + len(sentinelSuffix)
		sentinels = append(sentinels, s[start:end])
		i = end
	}
	return sentinels
}
