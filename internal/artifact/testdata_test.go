package artifact

import (
	"os"
	"path/filepath"
	"testing"
)

func readFixture(t *testing.T, rel string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", rel))
	if err != nil {
		t.Fatalf("read %s: %v", rel, err)
	}
	return b
}

func TestFixturesOK(t *testing.T) {
	tests := []struct {
		name   string
		rel    string
		parse  func(string, []byte) (Artifact, *ParseError)
		assert func(t *testing.T, a Artifact)
	}{
		{
			name: "claudemd basic",
			rel:  "ok/claudemd/basic.md",
			parse: func(p string, b []byte) (Artifact, *ParseError) {
				c, err := ParseClaudeMD(p, b)
				return c, err
			},
			assert: func(t *testing.T, a Artifact) {
				t.Helper()
				c, ok := a.(*ClaudeMD)
				if !ok {
					t.Fatalf("type = %T, want *ClaudeMD", a)
				}
				if !c.Frontmatter.Block.IsZero() {
					t.Errorf("basic CLAUDE.md should have no frontmatter")
				}
			},
		},
		{
			name: "skill writer",
			rel:  "ok/skills/writer.md",
			parse: func(p string, b []byte) (Artifact, *ParseError) {
				s, err := ParseSkill(p, b)
				return s, err
			},
			assert: func(t *testing.T, a Artifact) {
				t.Helper()
				s := a.(*Skill)
				if s.Name != "writer" {
					t.Errorf("Name = %q", s.Name)
				}
				if len(s.AllowedTools) != 2 {
					t.Errorf("AllowedTools = %v", s.AllowedTools)
				}
				// Byte-offset check: 'name' key starts at file line 2.
				if r := s.Frontmatter.KeyRange("name"); r.Start.Line != 2 || r.Start.Column != 1 {
					t.Errorf("name key range = %+v, want line 2 col 1", r.Start)
				}
			},
		},
		{
			name: "command review",
			rel:  "ok/commands/review.md",
			parse: func(p string, b []byte) (Artifact, *ParseError) {
				c, err := ParseCommand(p, b)
				return c, err
			},
			assert: func(t *testing.T, a Artifact) {
				t.Helper()
				c := a.(*Command)
				if c.Description != "review a pull request" {
					t.Errorf("Description = %q", c.Description)
				}
				if c.ArgumentHint != "<pr-number>" {
					t.Errorf("ArgumentHint = %q", c.ArgumentHint)
				}
			},
		},
		{
			name: "agent scribe",
			rel:  "ok/agents/scribe.md",
			parse: func(p string, b []byte) (Artifact, *ParseError) {
				a, err := ParseAgent(p, b)
				return a, err
			},
			assert: func(t *testing.T, a Artifact) {
				t.Helper()
				ag := a.(*Agent)
				if ag.Name != "scribe" {
					t.Errorf("Name = %q", ag.Name)
				}
			},
		},
		{
			name: "hook dedicated",
			rel:  "ok/hooks/dedicated.json",
			parse: func(p string, b []byte) (Artifact, *ParseError) {
				h, err := ParseHook(".claude/hooks/dedicated.json", b)
				return h, err
			},
			assert: func(t *testing.T, a Artifact) {
				t.Helper()
				h := a.(*Hook)
				if h.Embedded {
					t.Errorf("dedicated should not be Embedded")
				}
				if len(h.Entries) != 1 || h.Entries[0].Event != "PreToolUse" {
					t.Errorf("entries = %+v", h.Entries)
				}
			},
		},
		{
			name: "hook settings",
			rel:  "ok/hooks/settings.json",
			parse: func(p string, b []byte) (Artifact, *ParseError) {
				h, err := ParseHook(".claude/settings.json", b)
				return h, err
			},
			assert: func(t *testing.T, a Artifact) {
				t.Helper()
				h := a.(*Hook)
				if !h.Embedded {
					t.Errorf("settings should be Embedded")
				}
				if len(h.Entries) != 2 {
					t.Errorf("entries = %d, want 2", len(h.Entries))
				}
			},
		},
		{
			name: "plugin manifest",
			rel:  "ok/plugins/plugin.json",
			parse: func(p string, b []byte) (Artifact, *ParseError) {
				pl, err := ParsePlugin(p, b)
				return pl, err
			},
			assert: func(t *testing.T, a Artifact) {
				t.Helper()
				p := a.(*Plugin)
				if p.Name != "example" || p.Version != "1.2.3" {
					t.Errorf("name/version = %q/%q", p.Name, p.Version)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src := readFixture(t, tt.rel)
			a, perr := tt.parse(tt.rel, src)
			if perr != nil {
				t.Fatalf("parse = %v, want nil", perr)
			}
			tt.assert(t, a)
		})
	}
}

func TestFixturesBad(t *testing.T) {
	tests := []struct {
		name  string
		rel   string
		parse func(string, []byte) *ParseError
	}{
		{
			name: "unterminated_frontmatter",
			rel:  "bad/unterminated_frontmatter.md",
			parse: func(p string, b []byte) *ParseError {
				_, err := ParseSkill(p, b)
				return err
			},
		},
		{
			name: "invalid_yaml",
			rel:  "bad/invalid_yaml.md",
			parse: func(p string, b []byte) *ParseError {
				_, err := ParseCommand(p, b)
				return err
			},
		},
		{
			name: "invalid_json",
			rel:  "bad/invalid_json.json",
			parse: func(p string, b []byte) *ParseError {
				_, err := ParseHook(".claude/hooks/x.json", b)
				return err
			},
		},
		{
			name: "empty_json",
			rel:  "bad/empty_json.json",
			parse: func(p string, b []byte) *ParseError {
				_, err := ParseHook(".claude/hooks/x.json", b)
				return err
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src := readFixture(t, tt.rel)
			perr := tt.parse(tt.rel, src)
			if perr == nil {
				t.Fatalf("expected *ParseError for %s, got nil", tt.rel)
			}
			if perr.Range.IsZero() {
				t.Errorf("ParseError range should be non-zero for %s", tt.rel)
			}
		})
	}
}
