package output

import "context"

// YAMLParser parses raw YAML bytes into one or more documents (supports multi-doc files with ---).
type YAMLParser interface {
	ParseMultiDoc(ctx context.Context, data []byte, filePath string) ([]map[string]any, error)
}
