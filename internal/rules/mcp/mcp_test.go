package mcp

import (
	"strings"
	"testing"

	"github.com/donaldgifford/claudelint/internal/artifact"
)

func TestCommandRequired(t *testing.T) {
	cases := []struct {
		name    string
		command string
		wantN   int
	}{
		{"present", "uvx", 0},
		{"missing", "", 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := &artifact.MCPServer{Name: "srv", Command: tc.command}
			got := (&commandRequired{}).Check(nil, s)
			if len(got) != tc.wantN {
				t.Errorf("got %d diagnostics, want %d", len(got), tc.wantN)
			}
		})
	}
}

func TestCommandExistsOnPath(t *testing.T) {
	cases := []struct {
		name    string
		command string
		wantN   int
	}{
		{"known runner uvx", "uvx", 0},
		{"known runner npx", "npx", 0},
		{"absolute path", "/usr/local/bin/weird", 0},
		{"path with slash", "./bin/x", 0},
		{"empty", "", 0},
		{"typo uvv", "uvv", 1},
		{"unknown runner", "rando-runner", 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := &artifact.MCPServer{Name: "srv", Command: tc.command}
			got := (&commandExistsOnPath{}).Check(nil, s)
			if len(got) != tc.wantN {
				t.Errorf("got %d diagnostics, want %d", len(got), tc.wantN)
			}
		})
	}
}

func TestNoSecretsInEnv(t *testing.T) {
	// A known-prefix string (OpenAI sk-...) well over the length floor.
	secret := "sk-abcdefghijklmnopqrstuvwxyz0123456789"
	cases := []struct {
		name  string
		env   map[string]string
		wantN int
	}{
		{"empty env", nil, 0},
		{"benign values", map[string]string{"DEBUG": "1", "HOST": "localhost"}, 0},
		{"contains secret", map[string]string{"OPENAI_API_KEY": secret}, 1},
		{"multiple secrets", map[string]string{"A": secret, "B": secret}, 2},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := &artifact.MCPServer{Name: "srv", Env: tc.env}
			got := (&noSecretsInEnv{}).Check(nil, s)
			if len(got) != tc.wantN {
				t.Errorf("got %d diagnostics, want %d (%v)", len(got), tc.wantN, got)
			}
		})
	}
}

func TestNoUnsafeShell(t *testing.T) {
	cases := []struct {
		name    string
		command string
		args    []string
		wantN   int
	}{
		{"uvx ok", "uvx", []string{"mcp"}, 0},
		{"bash without -c", "bash", []string{"script.sh"}, 0},
		{"bash -c", "bash", []string{"-c", "ls"}, 1},
		{"sh -c", "sh", []string{"-c", "ls"}, 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := &artifact.MCPServer{Name: "srv", Command: tc.command, Args: tc.args}
			got := (&noUnsafeShell{}).Check(nil, s)
			if len(got) != tc.wantN {
				t.Errorf("got %d, want %d", len(got), tc.wantN)
			}
		})
	}
}

func TestServerNameRequired(t *testing.T) {
	if d := (&serverNameRequired{}).Check(nil, &artifact.MCPServer{}); len(d) != 1 {
		t.Errorf("empty name: want 1, got %d", len(d))
	}
	if d := (&serverNameRequired{}).Check(nil, &artifact.MCPServer{Name: "x"}); len(d) != 0 {
		t.Errorf("named: want 0, got %v", d)
	}
}

func TestDisabledCommented(t *testing.T) {
	enabled := &artifact.MCPServer{Name: "x", Disabled: false}
	if d := (&disabledCommented{}).Check(nil, enabled); len(d) != 0 {
		t.Errorf("enabled: want 0, got %v", d)
	}
	disabled := &artifact.MCPServer{Name: "x", Disabled: true}
	if d := (&disabledCommented{}).Check(nil, disabled); len(d) != 1 {
		t.Errorf("disabled without context: want 1, got %v", d)
	}
}

// optCtx is a test-only rules.Context backed by an in-memory option map.
type optCtx struct{ opts map[string]any }

func (*optCtx) RuleID() string          { return "" }
func (c *optCtx) Option(key string) any { return c.opts[key] }
func (*optCtx) Logf(_ string, _ ...any) {}

func TestServerAllowlistInList(t *testing.T) {
	s := &artifact.MCPServer{Name: "github", Command: "uvx"}
	ctx := &optCtx{opts: map[string]any{"allowlist": []any{"github", "deepwiki"}}}
	if d := (&serverAllowlist{}).Check(ctx, s); len(d) != 0 {
		t.Errorf("server in allowlist should pass, got %v", d)
	}
}

func TestServerAllowlistNotInList(t *testing.T) {
	s := &artifact.MCPServer{Name: "rogue-server", Command: "uvx"}
	ctx := &optCtx{opts: map[string]any{"allowlist": []any{"github"}}}
	d := (&serverAllowlist{}).Check(ctx, s)
	if len(d) != 1 {
		t.Fatalf("expected 1 diagnostic, got %d", len(d))
	}
	if d[0].Message != `MCP server "rogue-server" is not in the configured allowlist` {
		t.Errorf("message = %q", d[0].Message)
	}
}

// TestServerAllowlistMissingOption is the "fail loud" branch: with no
// allowlist configured, the rule emits a configuration-error
// diagnostic per server so users notice the misconfig instead of
// getting a silent no-op. The engine has no rule-level
// `default-enabled = false` hook today; this is the lint-time
// substitute for load-time validation.
func TestServerAllowlistMissingOption(t *testing.T) {
	s := &artifact.MCPServer{Name: "github", Command: "uvx"}
	// nil allowlist mirrors the engine's view when the user has not
	// supplied a value (DefaultOptions returns nil for "allowlist").
	ctx := &optCtx{opts: map[string]any{"allowlist": nil}}
	d := (&serverAllowlist{}).Check(ctx, s)
	if len(d) != 1 {
		t.Fatalf("expected 1 diagnostic for missing allowlist, got %d", len(d))
	}
	if !strings.Contains(d[0].Message, `"allowlist" option`) {
		t.Errorf("message should reference the allowlist option, got %q", d[0].Message)
	}
}

// TestServerAllowlistEmptyList covers the explicit "block all" path:
// allowlist = [] in the user's config means every MCP server should
// fire, not "every server allowed".
func TestServerAllowlistEmptyList(t *testing.T) {
	s := &artifact.MCPServer{Name: "anything", Command: "uvx"}
	ctx := &optCtx{opts: map[string]any{"allowlist": []any{}}}
	d := (&serverAllowlist{}).Check(ctx, s)
	if len(d) != 1 {
		t.Fatalf("expected 1 diagnostic for empty allowlist, got %d", len(d))
	}
	if !strings.Contains(d[0].Message, "not in the configured allowlist") {
		t.Errorf("message = %q", d[0].Message)
	}
}

// TestServerAllowlistRangePointsAtServer asserts the diagnostic
// targets the offending server's name range — file-level (0,0)
// would be a regression of the same bug as issue #15.
func TestServerAllowlistRangePointsAtServer(t *testing.T) {
	src := []byte(`{"servers":{"rogue":{"command":"uvx","args":["bad"]}}}`)
	servers, perr := artifact.ParseMCPFile(".mcp.json", src)
	if perr != nil {
		t.Fatalf("ParseMCPFile: %v", perr)
	}
	if len(servers) != 1 {
		t.Fatalf("expected 1 server, got %d", len(servers))
	}
	ctx := &optCtx{opts: map[string]any{"allowlist": []any{"github"}}}
	d := (&serverAllowlist{}).Check(ctx, servers[0])
	if len(d) != 1 {
		t.Fatalf("expected 1 diagnostic, got %d", len(d))
	}
	if d[0].Range.IsZero() {
		t.Errorf("diagnostic Range is zero; should point at server name")
	}
}
