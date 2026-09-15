package upstream_test

import (
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/donaldgifford/claudelint/internal/upstream"
)

const snippetDir = "testdata/snippets"

// fixtureDigest runs every extractor against its committed golden
// snippet and returns the assembled digest.
//
// Extractors are driven directly rather than through Extract, because
// each fixture is one already-scoped section and several sources feed
// more than one section.
func fixtureDigest(t *testing.T) *upstream.Digest {
	t.Helper()

	d := upstream.NewDigest()

	for _, e := range upstream.Extractors() {
		section := readFixture(t, upstream.SnippetPath(e))
		if err := e.Extract(section, d); err != nil {
			t.Fatalf("%s: extract from fixture: %v", e.Section(), err)
		}
	}

	d.Normalize()

	return d
}

func readFixture(t *testing.T, rel string) string {
	t.Helper()

	raw, err := os.ReadFile(filepath.Join(snippetDir, filepath.FromSlash(rel)))
	if err != nil {
		t.Fatalf("read fixture %s: %v", rel, err)
	}

	return string(raw)
}

// TestFixtureAnchorsStillResolve checks that each fixture still carries
// the heading its extractor scopes to, so the golden files exercise the
// anchoring path and not just the parsing.
func TestFixtureAnchorsStillResolve(t *testing.T) {
	t.Parallel()

	for _, e := range upstream.Extractors() {
		anchor := e.Anchor()
		if anchor == nil {
			continue
		}

		t.Run(e.Section(), func(t *testing.T) {
			t.Parallel()

			fixture := readFixture(t, upstream.SnippetPath(e))
			if _, ok := upstream.Section([]byte(fixture), anchor); !ok {
				t.Errorf("fixture %s no longer contains the anchor %s",
					upstream.SnippetPath(e), anchor)
			}
		})
	}
}

func TestExtractSkills(t *testing.T) {
	t.Parallel()

	d := fixtureDigest(t)

	if got := len(d.Skills.Frontmatter); got != 20 {
		t.Errorf("skills.frontmatter has %d fields, want 20", got)
	}
	assertField(t, d.Skills.Frontmatter, "description", "recommended")
	assertField(t, d.Skills.Frontmatter, "allowed-tools", "no")
	assertField(t, d.Skills.Frontmatter, "name", "no")
	assertHasAll(t, "skills.frontmatter", fieldNames(d.Skills.Frontmatter),
		"argument-hint", "arguments", "compatibility", "context", "effort", "hooks",
		"license", "metadata", "model", "paths", "shell", "when_to_use")

	wantPortable := []string{"allowed-tools", "compatibility", "description", "license", "metadata", "name"}
	if !slices.Equal(d.Skills.PortableFields, wantPortable) {
		t.Errorf("skills.portable_fields = %q, want %q", d.Skills.PortableFields, wantPortable)
	}

	if got := len(d.Skills.Substitutions); got != 10 {
		t.Errorf("skills.substitutions has %d entries, want 10", got)
	}
	assertHasAll(t, "skills.substitutions", d.Skills.Substitutions,
		"$ARGUMENTS", "$ARGUMENTS[N]", "$N", "$name",
		"${CLAUDE_PLUGIN_DATA}", "${CLAUDE_PLUGIN_ROOT}", "${CLAUDE_PROJECT_DIR}",
		"${CLAUDE_SESSION_ID}", "${CLAUDE_SKILL_DIR}", "${CLAUDE_EFFORT}")
}

func TestExtractAgents(t *testing.T) {
	t.Parallel()

	d := fixtureDigest(t)

	if got := len(d.Agents.Frontmatter); got != 17 {
		t.Errorf("agents.frontmatter has %d fields, want 17", got)
	}
	assertField(t, d.Agents.Frontmatter, "name", "yes")
	assertField(t, d.Agents.Frontmatter, "description", "yes")
	assertField(t, d.Agents.Frontmatter, "experimental", "no")

	tests := []struct {
		field string
		want  []string
	}{
		{field: "color", want: []string{"blue", "cyan", "green", "orange", "pink", "purple", "red", "yellow"}},
		{field: "effort", want: []string{"high", "low", "max", "medium", "xhigh"}},
		{field: "memory", want: []string{"local", "project", "user"}},
		{
			field: "permissionMode",
			want: []string{
				"acceptEdits", "auto", "bypassPermissions",
				"default", "dontAsk", "manual", "plan",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.field, func(t *testing.T) {
			t.Parallel()

			if got := d.Agents.Enums[tt.field]; !slices.Equal(got, tt.want) {
				t.Errorf("agents.enums.%s = %q, want %q", tt.field, got, tt.want)
			}
		})
	}

	// The model and isolation cells give an example alongside the
	// accepted values. The extractor records every code span, as
	// DESIGN-0006 specifies, and acknowledged.json is where the extra is
	// recorded as deliberate.
	assertHasAll(t, "agents.enums.model", d.Agents.Enums["model"],
		"fable", "haiku", "inherit", "opus", "sonnet")
	assertHasAll(t, "agents.enums.isolation", d.Agents.Enums["isolation"], "worktree")
}

func TestExtractPlugins(t *testing.T) {
	t.Parallel()

	d := fixtureDigest(t)

	if got := len(d.Plugins.ManifestFields); got != 26 {
		t.Errorf("plugins.manifest_fields has %d fields, want 26", got)
	}
	assertField(t, d.Plugins.ManifestFields, "name", "yes")

	byGroup := map[string]int{}
	for _, f := range d.Plugins.ManifestFields {
		byGroup[f.Group]++
	}

	wantGroups := map[string]int{"required": 1, "metadata": 11, "component": 14}
	for group, want := range wantGroups {
		if byGroup[group] != want {
			t.Errorf("plugins.manifest_fields group %q has %d rows, want %d", group, byGroup[group], want)
		}
	}

	if got := d.Plugins.Locations["manifest"]; got != ".claude-plugin/plugin.json" {
		t.Errorf("plugins.locations[manifest] = %q", got)
	}
	if got := d.Plugins.Locations["hooks"]; got != "hooks/hooks.json" {
		t.Errorf("plugins.locations[hooks] = %q", got)
	}
	if got := len(d.Plugins.Locations); got != 13 {
		t.Errorf("plugins.locations has %d entries, want 13", got)
	}

	if got := len(d.Plugins.SubstitutionFields); got != 5 {
		t.Errorf("plugins.substitution_fields has %d entries, want 5", got)
	}
	if got := d.Plugins.SubstitutionFields["mcp stdio servers"]; !slices.Equal(got, []string{"args", "command", "env"}) {
		t.Errorf("substitution fields for stdio servers = %q", got)
	}
}

func TestExtractMarketplaces(t *testing.T) {
	t.Parallel()

	d := fixtureDigest(t)

	if got := len(d.Marketplace.Fields); got != 9 {
		t.Errorf("marketplace.fields has %d fields, want 9", got)
	}
	assertField(t, d.Marketplace.Fields, "name", "yes")
	assertField(t, d.Marketplace.Fields, "owner", "yes")
	assertField(t, d.Marketplace.Fields, "plugins", "yes")
	assertField(t, d.Marketplace.Fields, "renames", "no")

	if got := len(d.Marketplace.OwnerFields); got != 3 {
		t.Errorf("marketplace.owner_fields has %d fields, want 3", got)
	}
	if got := len(d.Marketplace.PluginEntryFields); got != 16 {
		t.Errorf("marketplace.plugin_entry_fields has %d fields, want 16", got)
	}
	assertField(t, d.Marketplace.PluginEntryFields, "source", "yes")
	assertField(t, d.Marketplace.PluginEntryFields, "strict", "no")

	wantKinds := []string{"archive", "command", "git-subdir", "github", "npm", "url"}
	gotKinds := make([]string, 0, len(d.Marketplace.Sources))
	for k := range d.Marketplace.Sources {
		gotKinds = append(gotKinds, k)
	}
	slices.Sort(gotKinds)

	if !slices.Equal(gotKinds, wantKinds) {
		t.Errorf("marketplace.sources kinds = %q, want %q", gotKinds, wantKinds)
	}

	shapes := []struct {
		kind     string
		required []string
		optional []string
	}{
		{kind: "archive", required: []string{"url"}, optional: []string{"sha256"}},
		{kind: "command", required: []string{"command"}, optional: []string{"mode", "timeout"}},
		{kind: "github", required: []string{"repo"}, optional: []string{"ref", "sha"}},
		{kind: "npm", required: []string{"package"}, optional: []string{"registry", "version"}},
		{kind: "git-subdir", required: []string{"path", "url"}, optional: []string{"ref", "sha"}},
		{kind: "url", required: []string{"url"}, optional: []string{"ref", "sha"}},
	}

	for _, s := range shapes {
		t.Run("source/"+s.kind, func(t *testing.T) {
			t.Parallel()

			got := d.Marketplace.Sources[s.kind]
			if !slices.Equal(got.Required, s.required) {
				t.Errorf("%s required = %q, want %q", s.kind, got.Required, s.required)
			}
			if !slices.Equal(got.Optional, s.optional) {
				t.Errorf("%s optional = %q, want %q", s.kind, got.Optional, s.optional)
			}
		})
	}

	if got := len(d.Marketplace.ReservedNames); got != 17 {
		t.Errorf("marketplace.reserved_names has %d names, want 17: %q", got, d.Marketplace.ReservedNames)
	}
	assertHasAll(t, "marketplace.reserved_names", d.Marketplace.ReservedNames,
		"agent-skills", "anthropic-marketplace", "claude-code-marketplace", "claude-tag-plugins", "healthcare")

	// The sentence after the list gives examples of names that are
	// blocked for impersonating an official marketplace. They are not
	// reserved names and must not be collected.
	for _, bad := range []string{"official-claude-plugins", "anthropic-plugins-v2"} {
		if slices.Contains(d.Marketplace.ReservedNames, bad) {
			t.Errorf("reserved names leaked past the list and picked up %q", bad)
		}
	}
}

func TestExtractHooks(t *testing.T) {
	t.Parallel()

	d := fixtureDigest(t)

	if got := len(d.Hooks.Events); got != 33 {
		t.Errorf("hooks.events has %d events, want 33", got)
	}
	assertHasAll(t, "hooks.events", d.Hooks.Events,
		"DirectoryAdded", "PostModelSwitch", "PreModelSwitch",
		"PreToolUse", "PostToolUse", "SessionStart", "SessionEnd")

	wantTypes := []string{"agent", "command", "http", "mcp_tool", "prompt"}
	if !slices.Equal(d.Hooks.Types, wantTypes) {
		t.Errorf("hooks.types = %q, want %q", d.Hooks.Types, wantTypes)
	}

	for _, key := range append([]string{"common"}, wantTypes...) {
		if len(d.Hooks.HandlerFields[key]) == 0 {
			t.Errorf("hooks.handler_fields has no rows for %q", key)
		}
	}
	assertField(t, d.Hooks.HandlerFields["common"], "type", "yes")
	assertField(t, d.Hooks.HandlerFields["common"], "timeout", "no")
	assertField(t, d.Hooks.HandlerFields["command"], "command", "yes")
	assertField(t, d.Hooks.HandlerFields["http"], "url", "yes")
	assertField(t, d.Hooks.HandlerFields["mcp_tool"], "server", "yes")
	assertField(t, d.Hooks.HandlerFields["prompt"], "prompt", "yes")

	wantByType := map[string]int{"agent": 60, "command": 600, "http": 600, "mcp_tool": 600, "prompt": 30}
	for typ, want := range wantByType {
		if got := d.Hooks.TimeoutDefaults.ByType[typ]; got != want {
			t.Errorf("timeout default for %q = %d, want %d", typ, got, want)
		}
	}

	wantByEvent := map[string]int{
		"MessageDisplay": 10, "PostModelSwitch": 30,
		"PreModelSwitch": 30, "UserPromptSubmit": 30,
	}
	for event, want := range wantByEvent {
		if got := d.Hooks.TimeoutDefaults.ByEvent[event]; got != want {
			t.Errorf("timeout override for %q = %d, want %d", event, got, want)
		}
	}
	if got := len(d.Hooks.TimeoutDefaults.ByEvent); got != len(wantByEvent) {
		t.Errorf("timeout overrides = %v, want exactly %v", d.Hooks.TimeoutDefaults.ByEvent, wantByEvent)
	}
}

func TestExtractToolsAndMCP(t *testing.T) {
	t.Parallel()

	d := fixtureDigest(t)

	if got := len(d.Tools.Builtin); got != 45 {
		t.Errorf("tools.builtin has %d tools, want 45", got)
	}
	assertHasAll(t, "tools.builtin", d.Tools.Builtin,
		"Agent", "AskUserQuestion", "Bash", "EnterPlanMode", "SendMessage",
		"Skill", "TaskCreate", "ToolSearch", "Workflow", "Write")

	// These four are in claudelint's KnownTools today but are no longer
	// in the reference table. Phase 2 reconciles that.
	for _, gone := range []string{"Task", "BashOutput", "KillShell", "MultiEdit"} {
		if slices.Contains(d.Tools.Builtin, gone) {
			t.Errorf("tools.builtin unexpectedly contains %q", gone)
		}
	}

	wantTransports := []string{"http", "sse", "stdio", "ws"}
	if !slices.Equal(d.MCP.Transports, wantTransports) {
		t.Errorf("mcp.transports = %q, want %q", d.MCP.Transports, wantTransports)
	}
	if len(d.MCP.ServerFields) == 0 {
		t.Error("mcp.server_fields is empty")
	}
}

func TestExtractSchemaStoreAndPortable(t *testing.T) {
	t.Parallel()

	d := fixtureDigest(t)

	if got := len(d.SchemaStore.PluginManifest.Properties); got != 22 {
		t.Errorf("schemastore plugin properties = %d, want 22", got)
	}
	if got := len(d.SchemaStore.PluginManifest.HookEvents); got != 29 {
		t.Errorf("schemastore plugin hook events = %d, want 29", got)
	}
	assertHasAll(t, "schemastore plugin hook types", d.SchemaStore.PluginManifest.HookTypes,
		"agent", "command", "http", "mcp_tool", "prompt")
	assertHasAll(t, "schemastore plugin mcp types", d.SchemaStore.PluginManifest.MCPTypes,
		"http", "sse", "stdio", "ws")

	if got := len(d.SchemaStore.Marketplace.Properties); got != 9 {
		t.Errorf("schemastore marketplace properties = %d, want 9", got)
	}
	if got := len(d.SchemaStore.Marketplace.PluginEntryProperties); got != 26 {
		t.Errorf("schemastore marketplace plugin entry properties = %d, want 26", got)
	}
	// The schema predates the archive and command source kinds; that gap
	// is what the disagreement block records.
	if got := d.SchemaStore.Marketplace.SourceKinds; !slices.Equal(got, []string{"git-subdir", "github", "npm", "url"}) {
		t.Errorf("schemastore marketplace source kinds = %q", got)
	}

	if got := len(d.SchemaStore.Settings.HookEvents); got != 31 {
		t.Errorf("schemastore settings hook events = %d, want 31", got)
	}
	assertHasAll(t, "schemastore settings default modes", d.SchemaStore.Settings.DefaultModes,
		"acceptEdits", "bypassPermissions", "default", "plan")

	wantPortable := []string{"allowed-tools", "compatibility", "description", "license", "metadata", "name"}
	if !slices.Equal(d.Portable.AllowedFields, wantPortable) {
		t.Errorf("portable.allowed_fields = %q, want %q", d.Portable.AllowedFields, wantPortable)
	}
	if !slices.Equal(d.Portable.AnthropicAllowedFields, wantPortable) {
		t.Errorf("portable.anthropic_allowed_fields = %q, want %q", d.Portable.AnthropicAllowedFields, wantPortable)
	}

	wantLimits := map[string]int{"name": 64, "description": 1024, "compatibility": 500}
	for field, want := range wantLimits {
		if got := d.Portable.Limits[field]; got != want {
			t.Errorf("portable.limits[%s] = %d, want %d", field, got, want)
		}
	}
	if d.Portable.NamePattern == "" || !strings.HasPrefix(d.Portable.NamePattern, "^[a-z0-9") {
		t.Errorf("portable.name_pattern = %q", d.Portable.NamePattern)
	}
	if len(d.Portable.DescriptionConstraints) == 0 {
		t.Error("portable.description_constraints is empty")
	}
}

func TestExtractChangelog(t *testing.T) {
	t.Parallel()

	d := fixtureDigest(t)

	if !regexp.MustCompile(`^\d+\.\d+\.\d+$`).MatchString(d.Meta.ClaudeCodeLatest) {
		t.Errorf("meta.claude_code_latest = %q, want a semantic version", d.Meta.ClaudeCodeLatest)
	}
}

func TestExtractFailsOnMissingAnchor(t *testing.T) {
	t.Parallel()

	pages := map[string][]byte{
		upstream.SrcDocsHooks: []byte("# Hooks\n\n## Something else\n\nNo events here.\n"),
	}

	_, err := upstream.Extract(pages, upstream.ExtractOptions{})
	if err == nil {
		t.Fatal("Extract succeeded with a page missing every anchor")
	}
	if !errors.Is(err, upstream.ErrAnchorNotFound) && !errors.Is(err, upstream.ErrMissingSource) {
		t.Fatalf("error = %v, want a missing anchor or missing source", err)
	}

	var ee *upstream.ExtractError
	if !errors.As(err, &ee) {
		t.Fatalf("error is %T, want *upstream.ExtractError", err)
	}
	if ee.Source == "" || ee.Section == "" {
		t.Errorf("ExtractError does not name the source and section: %+v", ee)
	}
}

func TestExtractorFailureModes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		section string
		wantErr error
	}{
		{
			name:    "anchor present but table gone",
			section: "## Hook events\n\nThe events moved elsewhere.\n",
			wantErr: upstream.ErrEmptySection,
		},
		{
			name:    "headings present",
			section: "## Hook events\n\n### PreToolUse\n",
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			d := upstream.NewDigest()

			var events upstream.Extractor
			for _, e := range upstream.Extractors() {
				if e.Section() == "hooks.events" {
					events = e
				}
			}
			if events == nil {
				t.Fatal("no hooks.events extractor")
			}

			err := events.Extract(tt.section, d)
			if tt.wantErr == nil {
				if err != nil {
					t.Fatalf("Extract: %v", err)
				}

				return
			}
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestExtractRejectsWrongHeaders(t *testing.T) {
	t.Parallel()

	const section = "### Frontmatter reference\n\n" +
		"| Property | Mandatory |\n| --- | --- |\n| `name` | No |\n"

	d := upstream.NewDigest()

	var skills upstream.Extractor
	for _, e := range upstream.Extractors() {
		if e.Section() == "skills.frontmatter" {
			skills = e
		}
	}
	if skills == nil {
		t.Fatal("no skills.frontmatter extractor")
	}

	err := skills.Extract(section, d)
	if !errors.Is(err, upstream.ErrHeaders) {
		t.Fatalf("error = %v, want ErrHeaders", err)
	}
}

func TestFloorRejectsAGuttedSection(t *testing.T) {
	t.Parallel()

	baseline := &upstream.Digest{Hooks: upstream.Hooks{Events: make([]string, 33)}}
	for i := range baseline.Hooks.Events {
		baseline.Hooks.Events[i] = "Event" + string(rune('A'+i%26)) + string(rune('0'+i/26))
	}
	baseline.Normalize()

	pages := fixturePages(t)

	// A hooks page whose events section keeps only three headings is a
	// structural break, not a spec change.
	pages[upstream.SrcDocsHooks] = []byte(strings.Replace(
		string(pages[upstream.SrcDocsHooks]),
		readFixture(t, "docs.hooks/hook-events.md"),
		"## Hook events\n\n### PreToolUse\n\n### PostToolUse\n\n### Stop\n",
		1,
	))

	_, err := upstream.Extract(pages, upstream.ExtractOptions{Baseline: baseline})
	if !errors.Is(err, upstream.ErrBelowFloor) {
		t.Fatalf("error = %v, want ErrBelowFloor", err)
	}
}

// sectionSeparator ends one concatenated fixture section before the
// next begins. Without a top-level heading between them, a section that
// runs to the end of its page would swallow the section appended after
// it.
const sectionSeparator = "\n# ---\n\n"

// fixturePages assembles a page set from the fixtures, concatenating
// the sections that come from one source.
func fixturePages(t *testing.T) map[string][]byte {
	t.Helper()

	pages := make(map[string][]byte)
	seen := make(map[string]bool)

	for _, e := range upstream.Extractors() {
		rel := upstream.SnippetPath(e)
		if seen[rel] {
			continue
		}
		seen[rel] = true

		section := readFixture(t, rel)
		if e.Anchor() == nil {
			pages[e.Source()] = []byte(section)

			continue
		}
		pages[e.Source()] = append(pages[e.Source()], []byte(sectionSeparator+section)...)
	}

	return pages
}

// TestExtractFromFixturePages runs the whole pipeline over pages
// assembled from the fixtures, which is the closest offline stand-in
// for a live run.
func TestExtractFromFixturePages(t *testing.T) {
	t.Parallel()

	d, err := upstream.Extract(fixturePages(t), upstream.ExtractOptions{})
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}

	if got := len(d.Hooks.Events); got != 33 {
		t.Errorf("hooks.events = %d, want 33", got)
	}
	if got := len(d.Tools.Builtin); got != 45 {
		t.Errorf("tools.builtin = %d, want 45", got)
	}
	if d.DigestVersion != upstream.DigestVersion {
		t.Errorf("digest_version = %d, want %d", d.DigestVersion, upstream.DigestVersion)
	}
}

func fieldNames(fields []upstream.Field) []string {
	out := make([]string, 0, len(fields))
	for _, f := range fields {
		out = append(out, f.Name)
	}

	return out
}

func assertField(t *testing.T, fields []upstream.Field, name, required string) {
	t.Helper()

	for _, f := range fields {
		if f.Name != name {
			continue
		}
		if f.Required != required {
			t.Errorf("field %q required = %q, want %q", name, f.Required, required)
		}

		return
	}

	t.Errorf("field %q is missing from %v", name, fieldNames(fields))
}

func assertHasAll(t *testing.T, what string, got []string, want ...string) {
	t.Helper()

	for _, w := range want {
		if !slices.Contains(got, w) {
			t.Errorf("%s is missing %q; got %q", what, w, got)
		}
	}
}
