package schema

import (
	"strings"
	"testing"

	"github.com/donaldgifford/claudelint/internal/artifact"
	"github.com/donaldgifford/claudelint/internal/rules"
)

func TestParseRuleRegistered(t *testing.T) {
	r := rules.Get("schema/parse")
	if r == nil {
		t.Fatal("schema/parse should be registered")
	}
	if len(r.AppliesTo()) != len(artifact.AllKinds()) {
		t.Errorf("schema/parse should apply to every kind")
	}
}

func TestFrontmatterRequiredSkillOK(t *testing.T) {
	src := []byte("---\nname: x\ndescription: y\n---\n")
	s, _ := artifact.ParseSkill("s.md", src)
	r := &frontmatterRequired{}
	if d := r.Check(nil, s); len(d) != 0 {
		t.Errorf("expected no diagnostics, got %v", d)
	}
}

func TestFrontmatterRequiredSkillMissingName(t *testing.T) {
	src := []byte("---\ndescription: y\n---\n")
	s, _ := artifact.ParseSkill("s.md", src)
	r := &frontmatterRequired{}
	d := r.Check(nil, s)
	if len(d) != 1 {
		t.Fatalf("expected 1 diagnostic, got %d", len(d))
	}
	if d[0].RuleID != "schema/frontmatter-required" {
		t.Errorf("RuleID = %q", d[0].RuleID)
	}
	// Skill diagnostics use best-practice phrasing that names the
	// runtime fallback (OQ2) rather than the plain missing-key form.
	if !strings.Contains(d[0].Message, "directory name") {
		t.Errorf("skill name message should cite the fallback, got %q", d[0].Message)
	}
}

func TestFrontmatterRequiredSkillMissingDescriptionPhrasing(t *testing.T) {
	src := []byte("---\nname: y\n---\n")
	s, _ := artifact.ParseSkill("s.md", src)
	d := (&frontmatterRequired{}).Check(nil, s)
	if len(d) != 1 {
		t.Fatalf("expected 1 diagnostic, got %d", len(d))
	}
	if !strings.Contains(d[0].Message, "when to invoke") {
		t.Errorf("skill description message should explain invocation impact, got %q", d[0].Message)
	}
}

func TestFrontmatterRequiredCommandMissingDescription(t *testing.T) {
	src := []byte("---\nargument-hint: <x>\n---\n")
	c, _ := artifact.ParseCommand("c.md", src)
	r := &frontmatterRequired{}
	d := r.Check(nil, c)
	if len(d) != 1 {
		t.Fatalf("expected 1 diagnostic, got %+v", d)
	}
}

func TestFrontmatterRequiredNoFrontmatter(t *testing.T) {
	src := []byte("# just a header\n")
	s, _ := artifact.ParseSkill("s.md", src)
	r := &frontmatterRequired{}
	d := r.Check(nil, s)
	if len(d) != 1 {
		t.Fatalf("expected 1 diagnostic for no-frontmatter skill")
	}
}
