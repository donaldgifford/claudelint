package upstream

import (
	"encoding/json"
	"fmt"
	"slices"
)

// The JSON Schema sources express the same facts three different ways:
// top-level "properties" objects, "enum" arrays, and "const" scalars
// buried inside anyOf branches. Rather than hard-code the deep paths,
// which have already moved once between SchemaStore regenerations, the
// helpers here address shallow paths directly and find deep values by
// what they contain.

// jsonKeywordEnum and jsonKeywordConst are the JSON Schema keywords the
// walkers look for.
const (
	jsonKeywordEnum  = "enum"
	jsonKeywordConst = "const"
)

// pathIndexPlaceholder replaces array indices when a path is normalised
// for matching, so a value's position inside an anyOf branch does not
// matter.
const pathIndexPlaceholder = "N"

// decodeJSONDocument parses a schema document.
func decodeJSONDocument(raw []byte, what string) (any, error) {
	var root any
	if err := json.Unmarshal(raw, &root); err != nil {
		return nil, fmt.Errorf("decode %s: %w", what, err)
	}

	return root, nil
}

// jsonAt resolves a chain of object keys.
func jsonAt(root any, keys ...string) (any, bool) {
	cur := root

	for _, k := range keys {
		obj, ok := cur.(map[string]any)
		if !ok {
			return nil, false
		}
		next, exists := obj[k]
		if !exists {
			return nil, false
		}
		cur = next
	}

	return cur, true
}

// jsonKeys returns the keys of the object at the given path.
func jsonKeys(root any, keys ...string) []string {
	node, ok := jsonAt(root, keys...)
	if !ok {
		return nil
	}

	obj, ok := node.(map[string]any)
	if !ok {
		return nil
	}

	out := make([]string, 0, len(obj))
	for k := range obj {
		out = append(out, k)
	}

	return out
}

// jsonStrings returns the string members of the array at the given
// path.
func jsonStrings(root any, keys ...string) []string {
	node, ok := jsonAt(root, keys...)
	if !ok {
		return nil
	}

	return stringsOf(node)
}

// jsonEnumsContaining returns the union of every enum array in the
// document that contains sentinel. Identifying an enum by a member it
// must have survives the path changes that a regenerated schema brings.
func jsonEnumsContaining(root any, sentinel string) []string {
	var out []string

	walkJSON(root, nil, func(path []string, node any) {
		if len(path) == 0 || path[len(path)-1] != jsonKeywordEnum {
			return
		}
		values := stringsOf(node)
		if slices.Contains(values, sentinel) {
			out = append(out, values...)
		}
	})

	return out
}

// jsonConstsAt returns every const string whose normalised path
// contains each of the given segments in order. Array indices in the
// path are normalised away.
func jsonConstsAt(root any, segments ...string) []string {
	var out []string

	walkJSON(root, nil, func(path []string, node any) {
		if len(path) == 0 || path[len(path)-1] != jsonKeywordConst {
			return
		}
		s, ok := node.(string)
		if !ok {
			return
		}
		if pathContains(path, segments) {
			out = append(out, s)
		}
	})

	return out
}

// pathContains reports whether every segment appears in path, in order.
func pathContains(path, segments []string) bool {
	i := 0
	for _, p := range path {
		if i < len(segments) && p == segments[i] {
			i++
		}
	}

	return i == len(segments)
}

// walkJSON visits every node, passing the path taken to reach it with
// array indices replaced by a placeholder.
func walkJSON(node any, path []string, visit func(path []string, node any)) {
	visit(path, node)

	switch v := node.(type) {
	case map[string]any:
		for k, child := range v {
			walkJSON(child, append(path, k), visit)
		}
	case []any:
		for _, child := range v {
			walkJSON(child, append(path, pathIndexPlaceholder), visit)
		}
	}
}

// stringsOf returns the string members of a JSON array, ignoring any
// member that is not a string.
func stringsOf(node any) []string {
	arr, ok := node.([]any)
	if !ok {
		return nil
	}

	out := make([]string, 0, len(arr))
	for _, item := range arr {
		if s, isString := item.(string); isString {
			out = append(out, s)
		}
	}

	return out
}
