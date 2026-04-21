package artifact

import (
	"bytes"
	"strings"
	"testing"

	"github.com/donaldgifford/claudelint/internal/diag"
)

func TestParseMarkdownNoFrontmatter(t *testing.T) {
	src := []byte("# Hello\n\njust markdown.\n")
	doc, err := parseMarkdown("CLAUDE.md", src)
	if err != nil {
		t.Fatalf("parseMarkdown = %v, want nil", err)
	}
	if !doc.Frontmatter.Block.IsZero() {
		t.Errorf("no-frontmatter doc should have zero Block range, got %+v", doc.Frontmatter.Block)
	}
	if doc.Body.Start.Offset != 0 || doc.Body.End.Offset != len(src) {
		t.Errorf("Body range = %+v, want 0..%d", doc.Body, len(src))
	}
}

func TestParseMarkdownFrontmatterKeysAndRanges(t *testing.T) {
	src := []byte("---\nname: writer\ndescription: write things\n---\n# body\n")

	doc, err := parseMarkdown("skills/writer/SKILL.md", src)
	if err != nil {
		t.Fatalf("parseMarkdown = %v, want nil", err)
	}
	if _, ok := doc.Frontmatter.Keys["name"]; !ok {
		t.Errorf("Keys missing 'name'; got %v", keysOf(doc.Frontmatter.Keys))
	}
	if _, ok := doc.Frontmatter.Keys["description"]; !ok {
		t.Errorf("Keys missing 'description'; got %v", keysOf(doc.Frontmatter.Keys))
	}

	// 'name' key starts on file line 2, column 1.
	nameRange := doc.Frontmatter.Keys["name"]
	if nameRange.Start.Line != 2 || nameRange.Start.Column != 1 {
		t.Errorf("name range start = %+v, want line 2 col 1", nameRange.Start)
	}
	// 'description' key starts on file line 3, column 1.
	descRange := doc.Frontmatter.Keys["description"]
	if descRange.Start.Line != 3 || descRange.Start.Column != 1 {
		t.Errorf("description range start = %+v, want line 3 col 1", descRange.Start)
	}
}

func TestParseSkillExtractsTypedFields(t *testing.T) {
	src := []byte(
		"---\n" +
			"name: writer\n" +
			"description: write things\n" +
			"model: sonnet\n" +
			"allowed-tools:\n" +
			"  - Read\n" +
			"  - Write\n" +
			"---\n# body\n")

	s, err := ParseSkill("skills/writer/SKILL.md", src)
	if err != nil {
		t.Fatalf("ParseSkill = %v, want nil", err)
	}
	if s.Name != "writer" {
		t.Errorf("Name = %q, want writer", s.Name)
	}
	if s.Description != "write things" {
		t.Errorf("Description = %q, want \"write things\"", s.Description)
	}
	if s.Model != "sonnet" {
		t.Errorf("Model = %q, want sonnet", s.Model)
	}
	if len(s.AllowedTools) != 2 || s.AllowedTools[0] != "Read" || s.AllowedTools[1] != "Write" {
		t.Errorf("AllowedTools = %v, want [Read Write]", s.AllowedTools)
	}
	if s.Kind() != KindSkill {
		t.Errorf("Kind = %q, want %q", s.Kind(), KindSkill)
	}
}

func TestParseCommandAndAgent(t *testing.T) {
	cmdSrc := []byte(
		"---\ndescription: review\nargument-hint: <pr>\nallowed-tools: Read\n---\n")
	c, perr := ParseCommand(".claude/commands/review.md", cmdSrc)
	if perr != nil {
		t.Fatalf("ParseCommand = %v", perr)
	}
	if c.Description != "review" || c.ArgumentHint != "<pr>" || len(c.AllowedTools) != 1 {
		t.Errorf("Command = %+v", c)
	}

	agSrc := []byte("---\nname: scribe\ndescription: notes\ntools: [Read]\n---\n")
	a, perr := ParseAgent(".claude/agents/scribe.md", agSrc)
	if perr != nil {
		t.Fatalf("ParseAgent = %v", perr)
	}
	if a.Name != "scribe" || a.Description != "notes" || len(a.Tools) != 1 {
		t.Errorf("Agent = %+v", a)
	}
}

func TestParseMarkdownUnterminatedFrontmatter(t *testing.T) {
	src := []byte("---\nname: writer\nbody continues forever")
	_, perr := parseMarkdown("x", src)
	if perr == nil {
		t.Fatal("expected ParseError, got nil")
	}
	if !strings.Contains(perr.Message, "unterminated") {
		t.Errorf("message = %q, want contains 'unterminated'", perr.Message)
	}
	if perr.Range.IsZero() {
		t.Errorf("range should point at opening fence, got zero")
	}
}

func TestParseMarkdownInvalidYAML(t *testing.T) {
	src := []byte("---\nname: [unbalanced\n---\n")
	_, perr := parseMarkdown("x", src)
	if perr == nil {
		t.Fatal("expected ParseError, got nil")
	}
	if !strings.Contains(perr.Message, "invalid YAML") {
		t.Errorf("message = %q, want contains 'invalid YAML'", perr.Message)
	}
}

func TestParseClaudeMDNoFrontmatter(t *testing.T) {
	src := []byte("# CLAUDE.md\n\ninstructions\n")
	c, perr := ParseClaudeMD("CLAUDE.md", src)
	if perr != nil {
		t.Fatalf("ParseClaudeMD = %v", perr)
	}
	if !c.Frontmatter.Block.IsZero() {
		t.Errorf("CLAUDE.md without frontmatter should have zero Block")
	}
	if !bytes.Equal(c.Source(), src) {
		t.Errorf("Source should equal input")
	}
}

func keysOf(m map[string]diag.Range) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
