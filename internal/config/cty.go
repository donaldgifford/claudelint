package config

import (
	"github.com/zclconf/go-cty/cty"
)

// ctyValueToMap converts a cty.Value object into a plain map[string]any
// so rules can read their options without depending on go-cty. The
// conversion preserves nesting: inner objects become map[string]any,
// inner tuples/lists become []any, and primitive types unwrap to
// their Go equivalents.
//
// Non-object inputs return an empty map; callers of Options expect an
// object shape and anything else is treated as "no options set".
func ctyValueToMap(v cty.Value) map[string]any {
	if v.IsNull() || !v.Type().IsObjectType() {
		return map[string]any{}
	}
	out := make(map[string]any, len(v.Type().AttributeTypes()))
	for k := range v.Type().AttributeTypes() {
		out[k] = ctyToGo(v.GetAttr(k))
	}
	return out
}

func ctyToGo(v cty.Value) any {
	if v.IsNull() {
		return nil
	}
	t := v.Type()
	switch {
	case t == cty.String:
		return v.AsString()
	case t == cty.Number:
		// Options that look like ints stay as int64; floats stay as
		// float64. Conversion failures fall through to the big.Float
		// representation so rules can log a precise error if they
		// want number precision guarantees.
		bf := v.AsBigFloat()
		if bf.IsInt() {
			i, _ := bf.Int64()
			return i
		}
		f, _ := bf.Float64()
		return f
	case t == cty.Bool:
		return v.True()
	case t.IsObjectType() || t.IsMapType():
		inner := make(map[string]any, v.LengthInt())
		for it := v.ElementIterator(); it.Next(); {
			k, val := it.Element()
			inner[k.AsString()] = ctyToGo(val)
		}
		return inner
	case t.IsTupleType() || t.IsListType() || t.IsSetType():
		inner := make([]any, 0, v.LengthInt())
		for it := v.ElementIterator(); it.Next(); {
			_, val := it.Element()
			inner = append(inner, ctyToGo(val))
		}
		return inner
	default:
		return v.GoString()
	}
}
