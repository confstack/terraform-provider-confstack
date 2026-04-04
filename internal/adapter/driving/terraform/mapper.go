package terraform

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// mapToTerraformDynamic converts a map[string]any to a types.Dynamic wrapping a types.Object.
func mapToTerraformDynamic(m map[string]any) (types.Dynamic, error) {
	v, _, err := mapToTerraformValue(m)
	if err != nil {
		return types.DynamicNull(), err
	}
	return types.DynamicValue(v), nil
}

// mapToTerraformValue recursively converts a Go value to an attr.Value + attr.Type pair.
func mapToTerraformValue(v any) (attr.Value, attr.Type, error) {
	if v == nil {
		return types.DynamicNull(), types.DynamicType, nil
	}

	switch val := v.(type) {
	case map[string]any:
		attrTypes := make(map[string]attr.Type, len(val))
		attrVals := make(map[string]attr.Value, len(val))
		for k, mv := range val {
			childVal, childType, err := mapToTerraformValue(mv)
			if err != nil {
				return nil, nil, fmt.Errorf("key %q: %w", k, err)
			}
			attrTypes[k] = childType
			attrVals[k] = childVal
		}
		objVal, diags := types.ObjectValue(attrTypes, attrVals)
		if diags.HasError() {
			return nil, nil, fmt.Errorf("creating object: %s", diags.Errors()[0].Detail())
		}
		return objVal, objVal.Type(context.Background()), nil

	case []any:
		if len(val) == 0 {
			tupleVal, diags := types.TupleValue([]attr.Type{}, []attr.Value{})
			if diags.HasError() {
				return nil, nil, fmt.Errorf("creating empty tuple: %s", diags.Errors()[0].Detail())
			}
			return tupleVal, tupleVal.Type(context.Background()), nil
		}
		elemTypes := make([]attr.Type, len(val))
		elemVals := make([]attr.Value, len(val))
		for i, item := range val {
			ev, et, err := mapToTerraformValue(item)
			if err != nil {
				return nil, nil, fmt.Errorf("list index %d: %w", i, err)
			}
			elemTypes[i] = et
			elemVals[i] = ev
		}
		tupleVal, diags := types.TupleValue(elemTypes, elemVals)
		if diags.HasError() {
			return nil, nil, fmt.Errorf("creating tuple: %s", diags.Errors()[0].Detail())
		}
		return tupleVal, tupleVal.Type(context.Background()), nil

	case string:
		sv := types.StringValue(val)
		return sv, types.StringType, nil

	case bool:
		bv := types.BoolValue(val)
		return bv, types.BoolType, nil

	case int:
		fv := types.Float64Value(float64(val))
		return fv, types.Float64Type, nil

	case int64:
		fv := types.Float64Value(float64(val))
		return fv, types.Float64Type, nil

	case float64:
		fv := types.Float64Value(val)
		return fv, types.Float64Type, nil

	default:
		// Fallback: convert to string representation
		sv := types.StringValue(fmt.Sprintf("%v", val))
		return sv, types.StringType, nil
	}
}
