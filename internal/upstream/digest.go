package upstream

import (
	"bytes"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
)

// DigestVersion is the schema version of the committed digest. Bump it
// only when the shape changes in a way an older reader cannot handle;
// the diff compares content, not versions.
const DigestVersion = 1

// Requiredness values as the upstream tables spell them, lowercased.
const (
	RequiredYes         = "yes"
	RequiredNo          = "no"
	RequiredRecommended = "recommended"
)

// Digest is the extracted, committed view of the upstream Claude Code
// and Agent Skills specifications.
//
// Fields are declared in alphabetical order of their json tag. Go emits
// struct fields in declaration order and sorts only map keys, so
// declaration order is the only way to make the committed file
// key-sorted from top to bottom.
type Digest struct {
	Agents        Agents         `json:"agents"`
	DigestVersion int            `json:"digest_version"`
	Disagreements []Disagreement `json:"disagreements"`
	Hooks         Hooks          `json:"hooks"`
	Marketplace   Marketplace    `json:"marketplace"`
	MCP           MCP            `json:"mcp"`
	Meta          Meta           `json:"meta"`
	Plugins       Plugins        `json:"plugins"`
	Portable      Portable       `json:"portable"`
	SchemaStore   SchemaStore    `json:"schemastore"`
	Skills        Skills         `json:"skills"`
	Tools         Tools          `json:"tools"`
}

// Field is one row of a documented field table. Required is "yes",
// "no", or "recommended"; Type and Group carry the extra columns where
// the source table has them.
type Field struct {
	Group    string `json:"group,omitempty"`
	Name     string `json:"name"`
	Required string `json:"required,omitempty"`
	Type     string `json:"type,omitempty"`
}

// Meta is reporting metadata. It is informational: a change here never
// counts as drift.
type Meta struct {
	// ClaudeCodeLatest is the newest release in the Claude Code
	// changelog, without a leading "v".
	ClaudeCodeLatest string `json:"claude_code_latest"`
	// DocsMaxMarker is the highest "v2.N.N" marker seen across the
	// documentation pages.
	DocsMaxMarker string `json:"docs_max_marker"`
}

// Tools is the built-in tool reference.
type Tools struct {
	// Builtin is every tool name in the reference table.
	Builtin []string `json:"builtin"`
	// Deprecated is the subset of Builtin whose description cell begins
	// with "Deprecated". It is a separate array rather than a flag on a
	// row object so Builtin stays a plain string set and diffs as one.
	Deprecated []string `json:"deprecated"`
}

// Hooks is the hook configuration surface.
type Hooks struct {
	// Events is every documented hook event name.
	Events []string `json:"events"`
	// HandlerFields maps "common" and each handler type to its
	// documented fields.
	HandlerFields map[string][]Field `json:"handler_fields"`
	// TimeoutDefaults is the documented timeout in seconds.
	TimeoutDefaults TimeoutDefaults `json:"timeout_defaults"`
	// Types is every documented handler type.
	Types []string `json:"types"`
}

// TimeoutDefaults holds documented hook timeouts in seconds. The values
// are int rather than any so the digest never round-trips them through
// float64, and carry no omitempty so a documented zero survives.
type TimeoutDefaults struct {
	// ByEvent is the per-event override, where the documentation lowers
	// the default for a specific event.
	ByEvent map[string]int `json:"by_event"`
	// ByType is the default per handler type.
	ByType map[string]int `json:"by_type"`
}

// Skills is the skill frontmatter surface.
type Skills struct {
	// Frontmatter is every documented SKILL.md frontmatter field.
	Frontmatter []Field `json:"frontmatter"`
	// PortableFields is the subset usable outside Claude Code.
	PortableFields []string `json:"portable_fields"`
	// Substitutions is every documented string substitution.
	Substitutions []string `json:"substitutions"`
}

// Agents is the subagent frontmatter surface.
type Agents struct {
	// Enums maps a frontmatter field to the values its description cell
	// spells out. The extractor takes every inline code span in that
	// cell, so a cell that also gives an example carries that example
	// here; acknowledged.json is where such an item is recorded as
	// deliberate.
	Enums map[string][]string `json:"enums"`
	// Frontmatter is every documented subagent frontmatter field.
	Frontmatter []Field `json:"frontmatter"`
}

// Plugins is the plugin manifest surface.
type Plugins struct {
	// Locations maps a plugin component to its default path.
	Locations map[string]string `json:"locations"`
	// ManifestFields is every documented plugin.json field, grouped as
	// required, metadata, or component.
	ManifestFields []Field `json:"manifest_fields"`
	// SubstitutionFields maps a plugin component to the fields in which
	// path placeholders resolve.
	SubstitutionFields map[string][]string `json:"substitution_fields"`
}

// Marketplace is the marketplace manifest surface.
type Marketplace struct {
	// Fields is every documented top-level marketplace.json field.
	Fields []Field `json:"fields"`
	// OwnerFields is every documented field of the owner object.
	OwnerFields []Field `json:"owner_fields"`
	// PluginEntryFields is every documented field of a plugin entry.
	PluginEntryFields []Field `json:"plugin_entry_fields"`
	// ReservedNames is every marketplace name reserved for Anthropic.
	ReservedNames []string `json:"reserved_names"`
	// Sources maps a plugin source kind to its documented field split.
	Sources map[string]SourceShape `json:"sources"`
}

// SourceShape is the documented field split for one marketplace plugin
// source kind.
type SourceShape struct {
	// Optional is the fields the documentation marks with a trailing
	// question mark.
	Optional []string `json:"optional"`
	// Required is the fields it does not.
	Required []string `json:"required"`
}

// MCP is the Model Context Protocol server surface.
type MCP struct {
	// ServerFields is the documented shape of one server entry.
	ServerFields []Field `json:"server_fields"`
	// Transports is every documented transport type.
	Transports []string `json:"transports"`
}

// Portable is the Agent Skills specification, kept separate from the
// Claude Code superset so a future portability rule can read it alone.
type Portable struct {
	// AllowedFields is the specification's allowlist.
	AllowedFields []string `json:"allowed_fields"`
	// AnthropicAllowedFields is the allowlist Anthropic's own
	// quick_validate.py enforces.
	AnthropicAllowedFields []string `json:"anthropic_allowed_fields"`
	// DescriptionConstraints is the constraints quick_validate.py puts
	// on a description, one short phrase each.
	DescriptionConstraints []string `json:"description_constraints"`
	// Limits maps a field to its maximum length in characters.
	Limits map[string]int `json:"limits"`
	// NamePattern is the regular expression a skill name must match.
	NamePattern string `json:"name_pattern"`
}

// SchemaStore is the cross-check tier. Everything under it is
// informational: the published documentation wins where they disagree.
type SchemaStore struct {
	Marketplace    SchemaStoreMarketplace `json:"marketplace"`
	PluginManifest SchemaStorePlugin      `json:"plugin_manifest"`
	Settings       SchemaStoreSettings    `json:"settings"`
}

// SchemaStorePlugin is the claude-code-plugin-manifest schema.
type SchemaStorePlugin struct {
	HookEvents []string `json:"hook_events"`
	HookTypes  []string `json:"hook_types"`
	MCPTypes   []string `json:"mcp_types"`
	Properties []string `json:"properties"`
}

// SchemaStoreMarketplace is the claude-code-marketplace schema.
type SchemaStoreMarketplace struct {
	PluginEntryProperties []string `json:"plugin_entry_properties"`
	Properties            []string `json:"properties"`
	SourceKinds           []string `json:"source_kinds"`
}

// SchemaStoreSettings is the claude-code-settings schema.
type SchemaStoreSettings struct {
	DefaultModes []string `json:"default_modes"`
	HookEvents   []string `json:"hook_events"`
}

// Disagreement records one set difference between the documentation and
// a cross-check source.
type Disagreement struct {
	// DocsOnly is the items the documentation has and the source does
	// not.
	DocsOnly []string `json:"docs_only"`
	// Source is the cross-check source id.
	Source string `json:"source"`
	// SourceOnly is the items the source has and the documentation does
	// not.
	SourceOnly []string `json:"source_only"`
	// Topic is the digest path being compared, such as "hooks.events".
	Topic string `json:"topic"`
}

// NewDigest returns an empty digest with every map and slice
// initialised, so encoding it never emits a JSON null.
func NewDigest() *Digest {
	d := &Digest{DigestVersion: DigestVersion}
	d.Normalize()

	return d
}

// Normalize sorts and deduplicates every set-valued array, sorts every
// row array by its identity, and replaces nil slices and maps with
// empty ones. It is idempotent, and Encode calls it on a copy.
//
// A nil slice marshals as null and an empty one as [], so an extractor
// that legitimately finds nothing would otherwise flip the committed
// bytes. For the same reason no slice or map field in Digest carries
// omitempty.
func (d *Digest) Normalize() {
	d.Agents.Enums = normalizeStringSets(d.Agents.Enums)
	d.Agents.Frontmatter = normalizeFields(d.Agents.Frontmatter)

	d.Disagreements = normalizeDisagreements(d.Disagreements)

	d.Hooks.Events = normalizeSet(d.Hooks.Events)
	d.Hooks.Types = normalizeSet(d.Hooks.Types)
	d.Hooks.HandlerFields = normalizeFieldSets(d.Hooks.HandlerFields)
	d.Hooks.TimeoutDefaults.ByEvent = emptyIfNil(d.Hooks.TimeoutDefaults.ByEvent)
	d.Hooks.TimeoutDefaults.ByType = emptyIfNil(d.Hooks.TimeoutDefaults.ByType)

	d.Marketplace.Fields = normalizeFields(d.Marketplace.Fields)
	d.Marketplace.OwnerFields = normalizeFields(d.Marketplace.OwnerFields)
	d.Marketplace.PluginEntryFields = normalizeFields(d.Marketplace.PluginEntryFields)
	d.Marketplace.ReservedNames = normalizeSet(d.Marketplace.ReservedNames)
	d.Marketplace.Sources = normalizeSources(d.Marketplace.Sources)

	d.MCP.ServerFields = normalizeFields(d.MCP.ServerFields)
	d.MCP.Transports = normalizeSet(d.MCP.Transports)

	d.Plugins.Locations = emptyIfNil(d.Plugins.Locations)
	d.Plugins.ManifestFields = normalizeFields(d.Plugins.ManifestFields)
	d.Plugins.SubstitutionFields = normalizeStringSets(d.Plugins.SubstitutionFields)

	d.Portable.AllowedFields = normalizeSet(d.Portable.AllowedFields)
	d.Portable.AnthropicAllowedFields = normalizeSet(d.Portable.AnthropicAllowedFields)
	d.Portable.DescriptionConstraints = normalizeSet(d.Portable.DescriptionConstraints)
	d.Portable.Limits = emptyIfNil(d.Portable.Limits)

	d.normalizeSchemaStore()

	d.Skills.Frontmatter = normalizeFields(d.Skills.Frontmatter)
	d.Skills.PortableFields = normalizeSet(d.Skills.PortableFields)
	d.Skills.Substitutions = normalizeSet(d.Skills.Substitutions)

	d.Tools.Builtin = normalizeSet(d.Tools.Builtin)
	d.Tools.Deprecated = normalizeSet(d.Tools.Deprecated)
}

func (d *Digest) normalizeSchemaStore() {
	d.SchemaStore.Marketplace.PluginEntryProperties = normalizeSet(d.SchemaStore.Marketplace.PluginEntryProperties)
	d.SchemaStore.Marketplace.Properties = normalizeSet(d.SchemaStore.Marketplace.Properties)
	d.SchemaStore.Marketplace.SourceKinds = normalizeSet(d.SchemaStore.Marketplace.SourceKinds)
	d.SchemaStore.PluginManifest.HookEvents = normalizeSet(d.SchemaStore.PluginManifest.HookEvents)
	d.SchemaStore.PluginManifest.HookTypes = normalizeSet(d.SchemaStore.PluginManifest.HookTypes)
	d.SchemaStore.PluginManifest.MCPTypes = normalizeSet(d.SchemaStore.PluginManifest.MCPTypes)
	d.SchemaStore.PluginManifest.Properties = normalizeSet(d.SchemaStore.PluginManifest.Properties)
	d.SchemaStore.Settings.DefaultModes = normalizeSet(d.SchemaStore.Settings.DefaultModes)
	d.SchemaStore.Settings.HookEvents = normalizeSet(d.SchemaStore.Settings.HookEvents)
}

// Encode returns the canonical digest bytes: key-sorted, two-space
// indent, LF, one trailing newline, no HTML escaping. It normalizes a
// copy, so the receiver is untouched.
func (d *Digest) Encode() ([]byte, error) {
	clone := *d
	clone.Normalize()

	var buf bytes.Buffer
	if err := encodeJSON(&buf, &clone); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// Decode parses digest bytes.
func Decode(data []byte) (*Digest, error) {
	var d Digest
	if err := json.Unmarshal(data, &d); err != nil {
		return nil, fmt.Errorf("decode digest: %w", err)
	}
	d.Normalize()

	return &d, nil
}

// Tree renders the digest as a generic map so the diff and the sanity
// floors can address it by dotted path. Numbers stay json.Number, so
// they format exactly as they do in the digest file.
func (d *Digest) Tree() (map[string]any, error) {
	raw, err := d.Encode()
	if err != nil {
		return nil, err
	}

	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()

	var tree map[string]any
	if err := dec.Decode(&tree); err != nil {
		return nil, fmt.Errorf("decode digest tree: %w", err)
	}

	return tree, nil
}

// At resolves a dotted digest path such as "hooks.events" against a
// tree. ok is false when any segment is missing.
func At(tree map[string]any, path string) (any, bool) {
	var cur any = tree

	for _, seg := range strings.Split(path, ".") {
		m, isMap := cur.(map[string]any)
		if !isMap {
			return nil, false
		}
		next, exists := m[seg]
		if !exists {
			return nil, false
		}
		cur = next
	}

	return cur, true
}

// Count returns the cardinality at a dotted digest path: the length of
// an array or of an object. ok is false for scalars and missing paths.
func Count(tree map[string]any, path string) (int, bool) {
	node, ok := At(tree, path)
	if !ok {
		return 0, false
	}

	switch v := node.(type) {
	case []any:
		return len(v), true
	case map[string]any:
		return len(v), true
	default:
		return 0, false
	}
}

func normalizeSet(in []string) []string {
	out := make([]string, 0, len(in))
	for _, s := range in {
		if s = strings.TrimSpace(s); s != "" {
			out = append(out, s)
		}
	}

	slices.Sort(out)

	return slices.Compact(out)
}

// normalizeFields sorts rows by name, then group, then type, because
// the plugin manifest tables carry the same field name under more than
// one group.
func normalizeFields(in []Field) []Field {
	out := make([]Field, 0, len(in))
	for _, f := range in {
		f.Name = strings.TrimSpace(f.Name)
		if f.Name != "" {
			out = append(out, f)
		}
	}

	slices.SortFunc(out, func(a, b Field) int {
		return strings.Compare(a.Name+"\x00"+a.Group+"\x00"+a.Type, b.Name+"\x00"+b.Group+"\x00"+b.Type)
	})

	return slices.Compact(out)
}

func normalizeStringSets(in map[string][]string) map[string][]string {
	out := make(map[string][]string, len(in))
	for k, v := range in {
		out[k] = normalizeSet(v)
	}

	return out
}

func normalizeFieldSets(in map[string][]Field) map[string][]Field {
	out := make(map[string][]Field, len(in))
	for k, v := range in {
		out[k] = normalizeFields(v)
	}

	return out
}

func normalizeSources(in map[string]SourceShape) map[string]SourceShape {
	out := make(map[string]SourceShape, len(in))
	for k, v := range in {
		out[k] = SourceShape{
			Optional: normalizeSet(v.Optional),
			Required: normalizeSet(v.Required),
		}
	}

	return out
}

func normalizeDisagreements(in []Disagreement) []Disagreement {
	out := make([]Disagreement, 0, len(in))
	for _, d := range in {
		d.DocsOnly = normalizeSet(d.DocsOnly)
		d.SourceOnly = normalizeSet(d.SourceOnly)
		if len(d.DocsOnly) == 0 && len(d.SourceOnly) == 0 {
			continue
		}
		out = append(out, d)
	}

	slices.SortFunc(out, func(a, b Disagreement) int {
		return strings.Compare(a.Topic+"\x00"+a.Source, b.Topic+"\x00"+b.Source)
	})

	return out
}

func emptyIfNil[K comparable, V any](m map[K]V) map[K]V {
	if m == nil {
		return make(map[K]V)
	}

	return m
}
