package yaml

import (
	"bytes"
	"context"
	"errors"
	"io"

	"github.com/confstack/terraform-provider-confstack/internal/domain"
	"gopkg.in/yaml.v3"
)

// Parser implements port/output.YAMLParser using gopkg.in/yaml.v3.
type Parser struct{}

// NewParser returns a new YAML multi-document Parser.
func NewParser() *Parser {
	return &Parser{}
}

// ParseMultiDoc parses YAML bytes as one or more documents separated by ---.
// Each document must decode to a map or be nil (empty doc). Returns one map per document.
func (p *Parser) ParseMultiDoc(ctx context.Context, data []byte, filePath string) ([]map[string]any, error) {
	if len(bytes.TrimSpace(data)) == 0 {
		return []map[string]any{{}}, nil
	}

	decoder := yaml.NewDecoder(bytes.NewReader(data))
	var docs []map[string]any

	for {
		var raw any
		err := decoder.Decode(&raw)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, &domain.ParseError{FilePath: filePath, Detail: err.Error(), Cause: err}
		}

		if raw == nil {
			docs = append(docs, map[string]any{})
			continue
		}

		m, err := toStringKeyedMap(raw, filePath)
		if err != nil {
			return nil, err
		}
		docs = append(docs, m)
	}

	if len(docs) == 0 {
		return []map[string]any{{}}, nil
	}
	return docs, nil
}

// toStringKeyedMap converts a parsed YAML value to map[string]any recursively,
// ensuring all map keys are strings.
func toStringKeyedMap(v any, filePath string) (map[string]any, error) {
	switch val := v.(type) {
	case map[string]any:
		result := make(map[string]any, len(val))
		for k, mv := range val {
			converted, err := convertValue(mv, filePath)
			if err != nil {
				return nil, err
			}
			result[k] = converted
		}
		return result, nil
	case map[any]any:
		result := make(map[string]any, len(val))
		for k, mv := range val {
			ks, ok := k.(string)
			if !ok {
				return nil, &domain.ParseError{FilePath: filePath, Detail: "non-string map key"}
			}
			converted, err := convertValue(mv, filePath)
			if err != nil {
				return nil, err
			}
			result[ks] = converted
		}
		return result, nil
	default:
		return nil, &domain.ParseError{FilePath: filePath, Detail: "top-level document is not a map"}
	}
}

// convertValue recursively normalizes a YAML-decoded value, converting map[any]any to map[string]any.
func convertValue(v any, filePath string) (any, error) {
	switch val := v.(type) {
	case map[string]any:
		result := make(map[string]any, len(val))
		for k, mv := range val {
			converted, err := convertValue(mv, filePath)
			if err != nil {
				return nil, err
			}
			result[k] = converted
		}
		return result, nil
	case map[any]any:
		result := make(map[string]any, len(val))
		for k, mv := range val {
			ks, ok := k.(string)
			if !ok {
				return nil, &domain.ParseError{FilePath: filePath, Detail: "non-string map key"}
			}
			converted, err := convertValue(mv, filePath)
			if err != nil {
				return nil, err
			}
			result[ks] = converted
		}
		return result, nil
	case []any:
		result := make([]any, len(val))
		for i, item := range val {
			converted, err := convertValue(item, filePath)
			if err != nil {
				return nil, err
			}
			result[i] = converted
		}
		return result, nil
	default:
		return val, nil
	}
}
