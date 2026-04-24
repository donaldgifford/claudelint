package artifact

import (
	"errors"
	"fmt"

	"github.com/buger/jsonparser"
)

// ParseMCPFile parses a standalone .mcp.json file into one MCPServer
// per entry in the top-level servers{} object. The returned slice is
// in JSON document order and may be empty (a file with "servers": {}
// is well-formed).
//
// Each emitted MCPServer carries the same Base so rules that want to
// surface diagnostics on a per-server range can resolve absolute
// positions via the shared source.
func ParseMCPFile(filePath string, src []byte) ([]*MCPServer, *ParseError) {
	if !isJSONFile(filePath) {
		base := NewBase(filePath, src)
		return nil, &ParseError{
			Path:    filePath,
			Range:   base.ResolveRange(0, len(src)),
			Message: ".mcp.json files must be JSON",
		}
	}
	if err := validateJSON(src); err != nil {
		base := NewBase(filePath, src)
		return nil, &ParseError{
			Path:    filePath,
			Range:   base.ResolveRange(0, len(src)),
			Message: fmt.Sprintf("invalid JSON: %s", err.Error()),
			Cause:   err,
		}
	}

	serversRaw, dt, serversEndAbs, err := jsonparser.Get(src, "servers")
	if err != nil {
		if errors.Is(err, jsonparser.KeyPathNotFoundError) {
			return nil, nil
		}
		base := NewBase(filePath, src)
		return nil, &ParseError{
			Path:    filePath,
			Range:   base.ResolveRange(0, len(src)),
			Message: fmt.Sprintf(".mcp.json: reading servers: %s", err.Error()),
			Cause:   err,
		}
	}
	if dt != jsonparser.Object {
		base := NewBase(filePath, src)
		return nil, &ParseError{
			Path:    filePath,
			Range:   base.ResolveRange(0, len(src)),
			Message: fmt.Sprintf(".mcp.json: servers must be an object, got %s", dt.String()),
		}
	}
	serversStartAbs := serversEndAbs - len(serversRaw)

	out := collectServers(filePath, src, serversRaw, serversStartAbs, false)
	return out, nil
}

// ParseMCPEmbedded extracts MCPServer entries from a plugin manifest.
// pluginPath names the enclosing plugin.json; src is its bytes.
// Returns nil when the manifest has no mcp.servers field.
//
// Embedded servers share the plugin manifest's bytes as their source,
// so ranges resolve against the plugin.json, not a synthetic file.
func ParseMCPEmbedded(pluginPath string, src []byte) ([]*MCPServer, error) {
	serversRaw, dt, serversEndAbs, err := jsonparser.Get(src, "mcp", "servers")
	if err != nil {
		if errors.Is(err, jsonparser.KeyPathNotFoundError) {
			return nil, nil
		}
		return nil, fmt.Errorf("plugin manifest mcp.servers: %w", err)
	}
	if dt != jsonparser.Object {
		return nil, fmt.Errorf("plugin manifest mcp.servers must be an object, got %s", dt.String())
	}
	serversStartAbs := serversEndAbs - len(serversRaw)

	return collectServers(pluginPath, src, serversRaw, serversStartAbs, true), nil
}

// collectServers walks the servers{} object and builds one MCPServer
// per entry. Malformed individual entries are skipped silently — a
// single bad entry should not prevent the other servers from being
// linted. Rules will still observe the bad entry's shape by re-parsing
// the source around its range if they care.
func collectServers(filePath string, src, serversRaw []byte, serversAbs int, embedded bool) []*MCPServer {
	var out []*MCPServer
	err := jsonparser.ObjectEach(serversRaw, func(key, value []byte, vdt jsonparser.ValueType, valueEndOff int) error {
		if vdt != jsonparser.Object {
			return nil
		}
		// ObjectEach's offset argument points past the value in the
		// enclosing buffer. value itself is a sub-slice; its absolute
		// start in src equals valueEndAbs - len(value).
		valueEndAbs := serversAbs + valueEndOff
		valueStartAbs := valueEndAbs - len(value)

		base := NewBase(filePath, src)
		server := &MCPServer{
			Base:     base,
			Name:     string(key),
			Embedded: embedded,
		}
		// The key itself isn't returned with an offset by ObjectEach;
		// approximate by pointing NameRange at the value's span. Rules
		// that want the key byte span can refine later.
		server.NameRange = base.ResolveRange(valueStartAbs, valueEndAbs)
		server.Command, server.CommandRange = stringFieldAt(value, valueStartAbs, &base, "command")
		server.Args = stringArrayField(value, "args")
		server.Env = stringMapField(value, "env")
		server.Disabled = boolField(value, "disabled")
		out = append(out, server)
		return nil
	})
	if err != nil && !errors.Is(err, jsonparser.KeyPathNotFoundError) {
		// Return what we have; malformed trailing entries are not fatal.
		return out
	}
	return out
}

// stringMapField returns a map of string→string for an object-valued
// key. Missing or non-object keys return nil. Non-string values
// inside the object are skipped (not coerced).
func stringMapField(src []byte, key string) map[string]string {
	raw, dt, _, err := jsonparser.Get(src, key)
	if err != nil || dt != jsonparser.Object {
		return nil
	}
	out := make(map[string]string)
	if err := jsonparser.ObjectEach(raw, func(k, v []byte, vdt jsonparser.ValueType, _ int) error {
		if vdt == jsonparser.String {
			out[string(k)] = string(v)
		}
		return nil
	}); err != nil {
		// A malformed env object just means "no env" for this server;
		// rules elsewhere will surface the shape error if it matters.
		return nil
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// boolField returns the bool value of a top-level boolean key, or
// false for a missing / non-boolean key.
func boolField(src []byte, key string) bool {
	raw, dt, _, err := jsonparser.Get(src, key)
	if err != nil || dt != jsonparser.Boolean {
		return false
	}
	return len(raw) == 4 // "true"
}
