package upstream_test

import (
	"bytes"
	"encoding/json"
	"slices"
	"strings"
	"testing"

	"github.com/donaldgifford/claudelint/internal/upstream"
)

func TestNewDigestEncodesWithoutNulls(t *testing.T) {
	t.Parallel()

	raw, err := upstream.NewDigest().Encode()
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}

	if bytes.Contains(raw, []byte("null")) {
		t.Errorf("an empty digest encoded a JSON null:\n%s", raw)
	}
	if !bytes.HasSuffix(raw, []byte("\n")) {
		t.Error("encoding has no trailing newline")
	}
	if bytes.Contains(raw, []byte("\r")) {
		t.Error("encoding contains a carriage return")
	}
	if !bytes.Contains(raw, []byte("\n  \"agents\": {")) {
		t.Errorf("encoding is not two-space indented:\n%s", raw)
	}
}

func TestEncodeIsDeterministic(t *testing.T) {
	t.Parallel()

	d := &upstream.Digest{
		DigestVersion: upstream.DigestVersion,
		Hooks: upstream.Hooks{
			Events: []string{"PreToolUse", "ConfigChange", "PreToolUse", "SessionEnd"},
			Types:  []string{"prompt", "command", "agent"},
			TimeoutDefaults: upstream.TimeoutDefaults{
				ByType:  map[string]int{"prompt": 30, "command": 600, "agent": 60},
				ByEvent: map[string]int{"MessageDisplay": 10},
			},
		},
		Tools: upstream.Tools{Builtin: []string{"Write", "Bash", "Agent"}},
		Agents: upstream.Agents{
			Enums: map[string][]string{"color": {"red", "blue", "red"}},
		},
	}

	first, err := d.Encode()
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	second, err := d.Encode()
	if err != nil {
		t.Fatalf("Encode again: %v", err)
	}
	if !bytes.Equal(first, second) {
		t.Error("two encodings of the same digest differ")
	}

	// Round-tripping through Decode must reproduce the same bytes.
	back, err := upstream.Decode(first)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	third, err := back.Encode()
	if err != nil {
		t.Fatalf("Encode round trip: %v", err)
	}
	if !bytes.Equal(first, third) {
		t.Errorf("round trip changed the bytes:\n%s\n---\n%s", first, third)
	}
}

func TestNormalizeSortsAndDeduplicates(t *testing.T) {
	t.Parallel()

	d := &upstream.Digest{
		Hooks:  upstream.Hooks{Events: []string{"PreToolUse", "ConfigChange", "PreToolUse", "  ", "SessionEnd"}},
		Tools:  upstream.Tools{Builtin: []string{"Write", "Bash", "Agent", "Bash"}},
		Skills: upstream.Skills{Frontmatter: []upstream.Field{{Name: "name"}, {Name: "allowed-tools"}, {Name: ""}}},
		Agents: upstream.Agents{Enums: map[string][]string{"color": {"red", "blue", "red"}}},
	}
	d.Normalize()

	if want := []string{"ConfigChange", "PreToolUse", "SessionEnd"}; !slices.Equal(d.Hooks.Events, want) {
		t.Errorf("Events = %q, want %q", d.Hooks.Events, want)
	}
	if want := []string{"Agent", "Bash", "Write"}; !slices.Equal(d.Tools.Builtin, want) {
		t.Errorf("Builtin = %q, want %q", d.Tools.Builtin, want)
	}
	if want := []string{"blue", "red"}; !slices.Equal(d.Agents.Enums["color"], want) {
		t.Errorf("Enums[color] = %q, want %q", d.Agents.Enums["color"], want)
	}
	if len(d.Skills.Frontmatter) != 2 {
		t.Fatalf("Frontmatter has %d rows, want 2 (the empty name is dropped)", len(d.Skills.Frontmatter))
	}
	if d.Skills.Frontmatter[0].Name != "allowed-tools" {
		t.Errorf("Frontmatter is not sorted: %v", d.Skills.Frontmatter)
	}

	// Idempotent.
	before, err := d.Encode()
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	d.Normalize()
	after, err := d.Encode()
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	if !bytes.Equal(before, after) {
		t.Error("Normalize is not idempotent")
	}
}

// TestNormalizeKeepsSameNameDifferentGroup guards the plugin manifest
// case, where one field name appears under two groups.
func TestNormalizeKeepsSameNameDifferentGroup(t *testing.T) {
	t.Parallel()

	d := &upstream.Digest{
		Plugins: upstream.Plugins{ManifestFields: []upstream.Field{
			{Name: "hooks", Group: "component", Type: "string"},
			{Name: "hooks", Group: "metadata", Type: "string"},
			{Name: "hooks", Group: "component", Type: "string"},
		}},
	}
	d.Normalize()

	if got := len(d.Plugins.ManifestFields); got != 2 {
		t.Fatalf("got %d rows, want 2 distinct group entries: %v", got, d.Plugins.ManifestFields)
	}
}

func TestEncodeDoesNotEscapeHTML(t *testing.T) {
	t.Parallel()

	d := &upstream.Digest{
		Portable: upstream.Portable{
			NamePattern:            "^[a-z0-9]+(-[a-z0-9]+)*$",
			DescriptionConstraints: []string{"no < or > characters"},
		},
	}

	raw, err := d.Encode()
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	if bytes.Contains(raw, []byte(`\u003c`)) || bytes.Contains(raw, []byte(`\u003e`)) {
		t.Errorf("encoding escaped angle brackets to \\u form:\n%s", raw)
	}
	if !bytes.Contains(raw, []byte("no < or > characters")) {
		t.Errorf("encoding lost the literal angle brackets:\n%s", raw)
	}
	if !bytes.Contains(raw, []byte(`"^[a-z0-9]+(-[a-z0-9]+)*$"`)) {
		t.Errorf("encoding mangled the name pattern:\n%s", raw)
	}
}

func TestNormalizeDropsEmptyDisagreements(t *testing.T) {
	t.Parallel()

	d := &upstream.Digest{Disagreements: []upstream.Disagreement{
		{Topic: "hooks.events", Source: "schemastore.plugin", DocsOnly: []string{"PreModelSwitch"}},
		{Topic: "mcp.transports", Source: "schemastore.plugin"},
	}}
	d.Normalize()

	if len(d.Disagreements) != 1 {
		t.Fatalf("got %d disagreements, want only the non-empty one: %v", len(d.Disagreements), d.Disagreements)
	}
	if d.Disagreements[0].Topic != "hooks.events" {
		t.Errorf("kept the wrong disagreement: %v", d.Disagreements[0])
	}
}

func TestTreeAndCount(t *testing.T) {
	t.Parallel()

	d := &upstream.Digest{
		Hooks: upstream.Hooks{
			Events:          []string{"PreToolUse", "PostToolUse"},
			TimeoutDefaults: upstream.TimeoutDefaults{ByType: map[string]int{"command": 600}},
		},
		Tools: upstream.Tools{Builtin: []string{"Bash"}},
	}

	tree, err := d.Tree()
	if err != nil {
		t.Fatalf("Tree: %v", err)
	}

	tests := []struct {
		path      string
		wantCount int
		wantOK    bool
	}{
		{path: "hooks.events", wantCount: 2, wantOK: true},
		{path: "tools.builtin", wantCount: 1, wantOK: true},
		{path: "hooks.timeout_defaults.by_type", wantCount: 1, wantOK: true},
		{path: "hooks.nonexistent", wantOK: false},
		{path: "digest_version", wantOK: false},
		{path: "tools.builtin.deeper", wantOK: false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			t.Parallel()

			got, ok := upstream.Count(tree, tt.path)
			if ok != tt.wantOK {
				t.Fatalf("Count(%q) ok = %v, want %v", tt.path, ok, tt.wantOK)
			}
			if ok && got != tt.wantCount {
				t.Errorf("Count(%q) = %d, want %d", tt.path, got, tt.wantCount)
			}
		})
	}
}

// TestTreeKeepsIntegersIntact pins that timeouts survive the generic
// walk as integers rather than becoming 6e+02.
func TestTreeKeepsIntegersIntact(t *testing.T) {
	t.Parallel()

	d := &upstream.Digest{Hooks: upstream.Hooks{
		TimeoutDefaults: upstream.TimeoutDefaults{ByType: map[string]int{"command": 600000}},
	}}

	tree, err := d.Tree()
	if err != nil {
		t.Fatalf("Tree: %v", err)
	}

	node, ok := upstream.At(tree, "hooks.timeout_defaults.by_type")
	if !ok {
		t.Fatal("by_type missing from the tree")
	}
	m, isMap := node.(map[string]any)
	if !isMap {
		t.Fatalf("by_type is %T, want a map", node)
	}
	num, isNum := m["command"].(json.Number)
	if !isNum {
		t.Fatalf("timeout is %T, want json.Number", m["command"])
	}
	if num.String() != "600000" {
		t.Errorf("timeout = %s, want 600000", num)
	}
}

func TestDecodeRejectsGarbage(t *testing.T) {
	t.Parallel()

	if _, err := upstream.Decode([]byte("not json")); err == nil {
		t.Error("Decode accepted invalid JSON")
	}
}

// TestEncodeKeysAreSorted walks the encoded digest and asserts that the
// top-level keys come out in order, which is what makes a re-render a
// no-op in git.
func TestEncodeKeysAreSorted(t *testing.T) {
	t.Parallel()

	raw, err := upstream.NewDigest().Encode()
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}

	var keys []string
	for _, line := range strings.Split(string(raw), "\n") {
		if !strings.HasPrefix(line, `  "`) {
			continue
		}
		key, _, found := strings.Cut(strings.TrimPrefix(line, `  "`), `"`)
		if found {
			keys = append(keys, key)
		}
	}

	if len(keys) == 0 {
		t.Fatal("found no top-level keys")
	}
	if !slices.IsSorted(keys) {
		t.Errorf("top-level keys are not sorted: %v", keys)
	}
}
