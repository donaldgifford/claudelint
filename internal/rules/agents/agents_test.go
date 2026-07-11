package agents

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/donaldgifford/claudelint/internal/artifact"
	"github.com/donaldgifford/claudelint/internal/rules"
)

// newAgent parses a minimal agent file with the given extra
// frontmatter lines so rule tests exercise the real parser path.
func newAgent(t *testing.T, extra string) *artifact.Agent {
	t.Helper()
	src := []byte("---\nname: helper\ndescription: d\n" + extra + "---\nbody\n")
	a, perr := artifact.ParseAgent(".claude/agents/helper.md", src)
	if perr != nil {
		t.Fatalf("ParseAgent = %v", perr)
	}
	return a
}

func TestModelValid(t *testing.T) {
	cases := []struct {
		name  string
		model string // empty = omit the key
		wantN int
	}{
		{"omitted means inherit", "", 0},
		{"alias sonnet", "sonnet", 0},
		{"alias fable", "fable", 0},
		{"inherit", "inherit", 0},
		{"full model ID", "claude-opus-4-8", 0},
		{"typo sonet", "sonet", 1},
		{"uppercase alias", "Sonnet", 1},
		{"bogus", "gpt-4", 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			extra := ""
			if tc.model != "" {
				extra = "model: " + tc.model + "\n"
			}
			a := newAgent(t, extra)
			d := (&modelValid{}).Check(nil, a)
			if len(d) != tc.wantN {
				t.Fatalf("got %d diagnostics, want %d (%v)", len(d), tc.wantN, d)
			}
			if tc.wantN == 1 {
				if !strings.Contains(d[0].Message, "sonnet, opus, haiku, fable, inherit") {
					t.Errorf("message should name the valid values, got %q", d[0].Message)
				}
				if d[0].Range.IsZero() {
					t.Errorf("diagnostic should anchor at the model key range")
				}
			}
		})
	}
}

func TestNameFormat(t *testing.T) {
	cases := []struct {
		name      string
		agentName string
		wantN     int
	}{
		{"simple", "scribe", 0},
		{"hyphenated", "code-reviewer", 0},
		{"uppercase", "Reviewer", 1},
		{"underscore", "code_reviewer", 1},
		{"digits", "agent2", 1},
		{"trailing hyphen", "reviewer-", 1},
		{"double hyphen", "code--reviewer", 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			src := []byte("---\nname: " + tc.agentName + "\ndescription: d\n---\nbody\n")
			a, perr := artifact.ParseAgent(".claude/agents/x.md", src)
			if perr != nil {
				t.Fatalf("ParseAgent = %v", perr)
			}
			d := (&nameFormat{}).Check(nil, a)
			if len(d) != tc.wantN {
				t.Errorf("got %d diagnostics, want %d (%v)", len(d), tc.wantN, d)
			}
		})
	}

	// Empty name is frontmatter-required's finding, not ours.
	empty := &artifact.Agent{}
	if d := (&nameFormat{}).Check(nil, empty); len(d) != 0 {
		t.Errorf("empty name should be skipped, got %v", d)
	}
}

func TestToolsKnown(t *testing.T) {
	cases := []struct {
		name  string
		extra string
		wantN int
	}{
		{"no lists", "", 0},
		{"valid names and patterns", "tools: Read, Grep, mcp__github, Bash(git diff:*)\n", 0},
		{"typo in tools", "tools: Read, Wrte\n", 1},
		{"typo in disallowedTools", "disallowedTools: Wrte\n", 1},
		{"typos in both", "tools: Frob\ndisallowedTools: Nicate\n", 2},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			a := newAgent(t, tc.extra)
			d := (&toolsKnown{}).Check(nil, a)
			if len(d) != tc.wantN {
				t.Fatalf("got %d diagnostics, want %d (%v)", len(d), tc.wantN, d)
			}
			for _, dd := range d {
				if !strings.Contains(dd.Message, "silently ignores") {
					t.Errorf("message should explain the silent-ignore behavior, got %q", dd.Message)
				}
				if dd.Range.IsZero() {
					t.Errorf("diagnostic should anchor at the offending key range")
				}
			}
		})
	}
}

func TestPluginIgnoredFields(t *testing.T) {
	const ignored = "permissionMode: acceptEdits\nmcpServers:\n  - github\nhooks:\n  PreToolUse: []\n"

	// Plugin-distributed: every declared ignored field flags.
	a := newAgent(t, ignored)
	a.PluginDistributed = true
	d := (&pluginIgnoredFields{}).Check(nil, a)
	if len(d) != 3 {
		t.Fatalf("plugin agent: want 3 diagnostics, got %d (%v)", len(d), d)
	}
	for _, dd := range d {
		if !strings.Contains(dd.Message, "ignored for plugin-distributed") {
			t.Errorf("message = %q", dd.Message)
		}
		if dd.Range.IsZero() {
			t.Errorf("diagnostic should anchor at the offending key")
		}
	}

	// Same fields on a project-level agent: silent.
	proj := newAgent(t, ignored)
	if d := (&pluginIgnoredFields{}).Check(nil, proj); len(d) != 0 {
		t.Errorf("project agent should be silent, got %v", d)
	}

	// Plugin-distributed but clean: silent.
	clean := newAgent(t, "model: haiku\n")
	clean.PluginDistributed = true
	if d := (&pluginIgnoredFields{}).Check(nil, clean); len(d) != 0 {
		t.Errorf("clean plugin agent should be silent, got %v", d)
	}
}

func TestFieldEnums(t *testing.T) {
	cases := []struct {
		name  string
		extra string
		wantN int
	}{
		{"no enum fields", "", 0},
		{"all valid", "permissionMode: plan\neffort: high\ncolor: cyan\nisolation: worktree\nmemory: project\n", 0},
		{"manual alias accepted", "permissionMode: manual\n", 0},
		{"typo'd permissionMode", "permissionMode: acceptedits\n", 1},
		{"unknown effort", "effort: turbo\n", 1},
		{"unknown color", "color: magenta\n", 1},
		{"unknown isolation", "isolation: container\n", 1},
		{"unknown memory", "memory: global\n", 1},
		{"two bad fields", "effort: turbo\ncolor: magenta\n", 2},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			a := newAgent(t, tc.extra)
			d := (&fieldEnums{}).Check(nil, a)
			if len(d) != tc.wantN {
				t.Fatalf("got %d diagnostics, want %d (%v)", len(d), tc.wantN, d)
			}
			for _, dd := range d {
				if !strings.Contains(dd.Message, "want one of:") {
					t.Errorf("message should list the valid values, got %q", dd.Message)
				}
				if dd.Range.IsZero() {
					t.Errorf("diagnostic should anchor at the offending key range")
				}
			}
		})
	}
}

// optCtx stubs rules.Context for option-driven rule tests.
type optCtx struct{ opts map[string]any }

func (*optCtx) RuleID() string          { return "" }
func (c *optCtx) Option(key string) any { return c.opts[key] }
func (*optCtx) Logf(_ string, _ ...any) {}

func TestModelPolicy(t *testing.T) {
	cases := []struct {
		name    string
		extra   string // agent frontmatter beyond name/description
		opts    map[string]any
		wantN   int
		wantSub string // substring every diagnostic must contain
	}{
		{
			name:    "no options is a config error",
			opts:    map[string]any{},
			wantN:   1,
			wantSub: "enabled without options",
		},
		{
			name:    "both options is a config error",
			opts:    map[string]any{"require": "inherit", "allowlist": []any{"opus"}},
			wantN:   1,
			wantSub: "mutually exclusive",
		},
		{
			name:    "require only supports inherit",
			opts:    map[string]any{"require": "sonnet"},
			wantN:   1,
			wantSub: `only supports "inherit"`,
		},
		{
			name:    "allowlist must be strings",
			opts:    map[string]any{"allowlist": []any{42}},
			wantN:   1,
			wantSub: "must be a list of strings",
		},
		{
			name:    "invalid allowlist entry",
			opts:    map[string]any{"allowlist": []any{"gpt-4"}},
			wantN:   1,
			wantSub: "not a valid model reference",
		},
		{
			name:  "require inherit passes absent model",
			opts:  map[string]any{"require": "inherit"},
			wantN: 0,
		},
		{
			name:  "require inherit passes explicit inherit",
			extra: "model: inherit\n",
			opts:  map[string]any{"require": "inherit"},
			wantN: 0,
		},
		{
			name:    "require inherit flags declared model",
			extra:   "model: opus\n",
			opts:    map[string]any{"require": "inherit"},
			wantN:   1,
			wantSub: "violates the require",
		},
		{
			name:  "allowlist passes listed model",
			extra: "model: opus\n",
			opts:  map[string]any{"allowlist": []any{"opus", "inherit"}},
			wantN: 0,
		},
		{
			name:    "allowlist flags unlisted model",
			extra:   "model: haiku\n",
			opts:    map[string]any{"allowlist": []any{"opus"}},
			wantN:   1,
			wantSub: "not in the configured model allowlist",
		},
		{
			name:  "absent model evaluates as inherit",
			opts:  map[string]any{"allowlist": []any{"inherit"}},
			wantN: 0,
		},
		{
			name:    "absent model fires when inherit not allowed",
			opts:    map[string]any{"allowlist": []any{"opus"}},
			wantN:   1,
			wantSub: `"inherit" is not in the configured model allowlist`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			a := newAgent(t, tc.extra)
			d := (&modelPolicy{}).Check(&optCtx{opts: tc.opts}, a)
			if len(d) != tc.wantN {
				t.Fatalf("got %d diagnostics, want %d (%v)", len(d), tc.wantN, d)
			}
			for _, dd := range d {
				if tc.wantSub != "" && !strings.Contains(dd.Message, tc.wantSub) {
					t.Errorf("message = %q, want substring %q", dd.Message, tc.wantSub)
				}
				if dd.Range.IsZero() {
					t.Errorf("diagnostic should carry a non-zero range")
				}
			}
		})
	}
}

func TestModelPolicyIsOptIn(t *testing.T) {
	if !rules.IsOptIn(&modelPolicy{}) {
		t.Error("agents/model-policy must be opt-in")
	}
}

// TestFixtureSweep runs every rule in this package against the shared
// doc-valid full-field fixture (must stay silent) and a kitchen-sink
// invalid agent (every rule must fire with a usable range) — the
// Phase 4 success criterion in one place.
func TestFixtureSweep(t *testing.T) {
	allRules := []rules.Rule{
		&modelValid{}, &nameFormat{}, &toolsKnown{}, &fieldEnums{}, &pluginIgnoredFields{},
	}

	src, err := os.ReadFile(filepath.Join(
		"..", "..", "artifact", "testdata", "ok", "agents", "full.md"))
	if err != nil {
		t.Fatal(err)
	}
	valid, perr := artifact.ParseAgent(".claude/agents/full.md", src)
	if perr != nil {
		t.Fatalf("ParseAgent(full.md) = %v", perr)
	}
	for _, r := range allRules {
		if d := r.Check(nil, valid); len(d) != 0 {
			t.Errorf("%s on the valid fixture: %v", r.ID(), d)
		}
	}

	bad, perr := artifact.ParseAgent(".claude/agents/bad.md", []byte(
		"---\nname: Bad_Agent\ndescription: d\nmodel: sonet\ntools: WriteFil\n"+
			"permissionMode: yolo\nmcpServers:\n  - github\n---\nbody\n"))
	if perr != nil {
		t.Fatalf("ParseAgent(bad) = %v", perr)
	}
	bad.PluginDistributed = true
	for _, r := range allRules {
		d := r.Check(nil, bad)
		if len(d) == 0 {
			t.Errorf("%s did not fire on the kitchen-sink invalid agent", r.ID())
			continue
		}
		for _, dd := range d {
			if dd.Range.IsZero() {
				t.Errorf("%s emitted a zero range: %+v", r.ID(), dd)
			}
		}
	}
}

func TestModelValidRunsOnSkillsAndCommands(t *testing.T) {
	s, perr := artifact.ParseSkill("skills/x/SKILL.md",
		[]byte("---\nname: x\ndescription: d\nmodel: sonet\n---\nbody\n"))
	if perr != nil {
		t.Fatalf("ParseSkill = %v", perr)
	}
	if d := (&modelValid{}).Check(nil, s); len(d) != 1 {
		t.Errorf("skill with typo'd model: want 1, got %d", len(d))
	}

	c, perr := artifact.ParseCommand(".claude/commands/x.md",
		[]byte("---\ndescription: d\nmodel: haiku\n---\nbody\n"))
	if perr != nil {
		t.Fatalf("ParseCommand = %v", perr)
	}
	if d := (&modelValid{}).Check(nil, c); len(d) != 0 {
		t.Errorf("command with valid model: want 0, got %v", d)
	}
}
