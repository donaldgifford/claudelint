package upstream

import (
	"regexp"
	"sort"
)

// mcpTypeLiteral matches a transport declaration in the MCP page's JSON
// examples and prose, such as `"type": "http"`. The page documents the
// transports through those literals rather than a table, so the literal
// is the handle.
var mcpTypeLiteral = regexp.MustCompile(`"type"\s*:\s*"([a-z_][a-z0-9_-]*)"`)

// mcpServerKeys are the fields of one MCP server entry, read from the
// plugin manifest schema's mcpServers definition per DESIGN-0006 OQ5.
// Server shape is the one part of the MCP surface the schema states
// precisely and the prose does not.
var mcpServerKeys = []string{segMCPServers}

// mcpTransports reads the transport types from the MCP page.
//
// The whole page is in scope rather than one section: stdio is
// introduced by a bare command example, while the remote transports are
// shown under their own headings, so no single section holds all four.
type mcpTransports struct{}

var _ Extractor = mcpTransports{}

func (mcpTransports) Section() string        { return "mcp.transports" }
func (mcpTransports) Source() string         { return SrcDocsMCP }
func (mcpTransports) Anchor() *regexp.Regexp { return nil }

func (mcpTransports) Extract(section string, out *Digest) error {
	seen := make(map[string]struct{})
	for _, m := range mcpTypeLiteral.FindAllStringSubmatch(section, -1) {
		seen[m[1]] = struct{}{}
	}

	transports := make([]string, 0, len(seen))
	for t := range seen {
		transports = append(transports, t)
	}
	sort.Strings(transports)

	out.MCP.Transports = transports

	return nonEmpty(transports, "mcp transports")
}

// mcpServerFields reads the shape of one MCP server entry from the
// plugin manifest schema.
type mcpServerFields struct{}

var _ Extractor = mcpServerFields{}

func (mcpServerFields) Section() string        { return "mcp.server_fields" }
func (mcpServerFields) Source() string         { return SrcSchemaStorePlugin }
func (mcpServerFields) Anchor() *regexp.Regexp { return nil }

func (mcpServerFields) Extract(section string, out *Digest) error {
	root, err := decodeJSONDocument([]byte(section), "plugin manifest schema")
	if err != nil {
		return err
	}

	seen := make(map[string]struct{})

	walkJSON(root, nil, func(path []string, node any) {
		if len(path) == 0 || path[len(path)-1] != segProperties {
			return
		}
		if !pathContains(path, mcpServerKeys) {
			return
		}
		obj, ok := node.(map[string]any)
		if !ok {
			return
		}
		for k := range obj {
			seen[k] = struct{}{}
		}
	})

	fields := make([]Field, 0, len(seen))
	for name := range seen {
		fields = append(fields, Field{Name: name})
	}

	out.MCP.ServerFields = fields

	return nonEmpty(fields, "mcp server fields")
}
