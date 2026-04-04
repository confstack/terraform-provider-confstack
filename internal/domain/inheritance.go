package domain

import "fmt"

// InheritDirective describes a single template reference in an _inherit list.
type InheritDirective struct {
	Template string
	Except   []string
}

// CollectTemplates walks the merged tree and collects all templates from all _templates blocks.
// Template names must be globally unique; duplicates produce a DuplicateTemplateError.
func CollectTemplates(tree map[string]any, templatesKey, inheritKey string) (map[string]map[string]any, error) {
	templates := make(map[string]map[string]any)
	if err := collectTemplatesRecursive(tree, templatesKey, inheritKey, templates); err != nil {
		return nil, err
	}
	return templates, nil
}

func collectTemplatesRecursive(node map[string]any, templatesKey, inheritKey string, out map[string]map[string]any) error {
	if tmplBlock, ok := node[templatesKey]; ok {
		if tmplMap, ok := tmplBlock.(map[string]any); ok {
			for name, val := range tmplMap {
				if _, exists := out[name]; exists {
					return &DuplicateTemplateError{TemplateName: name}
				}
				tmpl, ok := val.(map[string]any)
				if !ok {
					// Non-map template body is not valid; treat as empty map
					tmpl = map[string]any{}
				}
				// Validate template does not contain _inherit
				if _, hasInherit := tmpl[inheritKey]; hasInherit {
					return &TemplateWithInheritError{
						TemplateName: name,
						InheritKey:   inheritKey,
					}
				}
				out[name] = tmpl
			}
		}
	}

	for _, v := range node {
		if child, ok := v.(map[string]any); ok {
			if err := collectTemplatesRecursive(child, templatesKey, inheritKey, out); err != nil {
				return err
			}
		}
	}
	return nil
}

// ResolveInheritance walks the tree and resolves all _inherit directives using bubble-up template lookup.
func ResolveInheritance(tree map[string]any, templatesKey, inheritKey string) (map[string]any, error) {
	// First collect all templates globally for uniqueness validation
	_, err := CollectTemplates(tree, templatesKey, inheritKey)
	if err != nil {
		return nil, err
	}

	// Resolve inheritance depth-first, passing parent templates via ancestor chain
	result, err := resolveNode(tree, templatesKey, inheritKey, nil, "")
	if err != nil {
		return nil, err
	}
	return result, nil
}

// resolveNode resolves inheritance within a map node. parentTemplates is the accumulated
// templates visible from ancestor scopes (for bubble-up lookup).
func resolveNode(node map[string]any, templatesKey, inheritKey string, parentTemplates map[string]map[string]any, nodePath string) (map[string]any, error) {
	// Build local templates for this scope (sibling _templates)
	localTemplates := make(map[string]map[string]any)
	for k, v := range parentTemplates {
		localTemplates[k] = v
	}
	if tmplBlock, ok := node[templatesKey]; ok {
		if tmplMap, ok := tmplBlock.(map[string]any); ok {
			for name, val := range tmplMap {
				tmpl, ok := val.(map[string]any)
				if !ok {
					tmpl = map[string]any{}
				}
				localTemplates[name] = tmpl
			}
		}
	}

	result := make(map[string]any, len(node))

	// Process each key in sorted order for determinism
	for _, k := range sortedKeys(node) {
		v := node[k]
		if k == templatesKey {
			// Will be stripped by cleanup; skip for now (we copy it for cleanup phase)
			result[k] = v
			continue
		}

		childMap, isMap := v.(map[string]any)
		if !isMap {
			result[k] = v
			continue
		}

		childPath := joinPath(nodePath, k)

		// Check if this child has an _inherit directive
		if inheritVal, hasInherit := childMap[inheritKey]; hasInherit {
			directives, err := parseInheritDirective(inheritVal)
			if err != nil {
				return nil, fmt.Errorf("entry %q: parsing _inherit: %w", childPath, err)
			}
			childMap, err = resolveInheritDirectives(childMap, inheritKey, childPath, directives, localTemplates)
			if err != nil {
				return nil, err
			}
		}

		// Recurse into child (now with or without _inherit resolved)
		resolved, err := resolveNode(childMap, templatesKey, inheritKey, localTemplates, childPath)
		if err != nil {
			return nil, err
		}
		result[k] = resolved
	}

	return result, nil
}

// resolveInheritDirectives builds the inherited base from directives and merges
// the entry's own values on top.
func resolveInheritDirectives(childMap map[string]any, inheritKey, childPath string, directives []InheritDirective, localTemplates map[string]map[string]any) (map[string]any, error) {
	inheritedBase := map[string]any{}
	for _, d := range directives {
		tmpl, ok := localTemplates[d.Template]
		if !ok {
			return nil, &TemplateNotFoundError{
				EntryPath:    childPath,
				TemplateName: d.Template,
			}
		}
		// Apply except filter
		filtered := applyExcept(tmpl, d.Except)
		merged, err := DeepMerge(inheritedBase, filtered, childPath, "_template:"+d.Template, childPath)
		if err != nil {
			return nil, fmt.Errorf("merging template %q for entry %q: %w", d.Template, childPath, err)
		}
		inheritedBase = merged
	}

	// Remove _inherit from child before merging child's own values on top
	childWithoutInherit := make(map[string]any, len(childMap))
	for ck, cv := range childMap {
		if ck == inheritKey {
			continue
		}
		childWithoutInherit[ck] = cv
	}

	// Merge: inheritedBase is the base, child's own values override
	result, err := DeepMerge(inheritedBase, childWithoutInherit, childPath, "_inherited", childPath)
	if err != nil {
		return nil, fmt.Errorf("applying overrides for entry %q: %w", childPath, err)
	}
	return result, nil
}

// parseInheritObject parses a single {template, except} map from an _inherit list.
func parseInheritObject(obj map[string]any) (InheritDirective, error) {
	tmplName, ok := obj["template"].(string)
	if !ok {
		return InheritDirective{}, fmt.Errorf("_inherit list object must have a string 'template' key")
	}
	var except []string
	if exceptVal, ok := obj["except"]; ok {
		if exceptList, ok := exceptVal.([]any); ok {
			for _, e := range exceptList {
				if es, ok := e.(string); ok {
					except = append(except, es)
				}
			}
		}
	}
	return InheritDirective{Template: tmplName, Except: except}, nil
}

// parseInheritDirective parses the _inherit value into a slice of InheritDirective.
// Supports: string, []string, []map with {template, except}.
func parseInheritDirective(val any) ([]InheritDirective, error) {
	switch v := val.(type) {
	case string:
		return []InheritDirective{{Template: v}}, nil
	case []any:
		var directives []InheritDirective
		for _, item := range v {
			switch iv := item.(type) {
			case string:
				directives = append(directives, InheritDirective{Template: iv})
			case map[string]any:
				d, err := parseInheritObject(iv)
				if err != nil {
					return nil, err
				}
				directives = append(directives, d)
			default:
				return nil, fmt.Errorf("_inherit list items must be strings or objects, got %T", item)
			}
		}
		return directives, nil
	default:
		return nil, fmt.Errorf("_inherit must be a string or list, got %T", val)
	}
}

// applyExcept returns a copy of m with the specified keys removed.
func applyExcept(m map[string]any, except []string) map[string]any {
	if len(except) == 0 {
		result := make(map[string]any, len(m))
		for k, v := range m {
			result[k] = v
		}
		return result
	}
	skip := make(map[string]bool, len(except))
	for _, e := range except {
		skip[e] = true
	}
	result := make(map[string]any, len(m))
	for k, v := range m {
		if !skip[k] {
			result[k] = v
		}
	}
	return result
}
