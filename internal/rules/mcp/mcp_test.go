package mcp

import (
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
