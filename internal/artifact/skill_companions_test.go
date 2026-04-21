package artifact

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIndexSkillCompanions(t *testing.T) {
	skillDir := t.TempDir()

	for _, p := range []string{
		"SKILL.md",
		"references/intro.md",
		"references/nested/deep.md",
		"scripts/run.sh",
		"templates/email.md",
		"unrelated.txt", // not in a canonical subdir → ignored
	} {
		full := filepath.Join(skillDir, p)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(full, []byte("x"), 0o644); err != nil {
			t.Fatalf("write: %v", err)
		}
	}

	s := &Skill{}
	if err := IndexSkillCompanions(s, skillDir); err != nil {
		t.Fatalf("IndexSkillCompanions = %v", err)
	}

	byPath := make(map[string]string, len(s.Companions))
	for _, c := range s.Companions {
		byPath[c.RelPath] = c.Kind
	}
	want := map[string]string{
		"references/intro.md":       "references",
		"references/nested/deep.md": "references",
		"scripts/run.sh":            "scripts",
		"templates/email.md":        "templates",
	}
	for p, wantKind := range want {
		if got, ok := byPath[p]; !ok || got != wantKind {
			t.Errorf("Companions[%q] = %q/ok=%v, want %q", p, got, ok, wantKind)
		}
	}
	if _, bad := byPath["unrelated.txt"]; bad {
		t.Errorf("Companions should not include unrelated files")
	}
}

func TestIndexSkillCompanionsMissingDirs(t *testing.T) {
	skillDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("x"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	s := &Skill{}
	if err := IndexSkillCompanions(s, skillDir); err != nil {
		t.Fatalf("IndexSkillCompanions = %v, want nil on missing subdirs", err)
	}
	if len(s.Companions) != 0 {
		t.Errorf("Companions = %v, want empty", s.Companions)
	}
}

func TestSkillCompanionsByKindAndHasPath(t *testing.T) {
	s := &Skill{Companions: []Companion{
		{RelPath: "references/intro.md", Kind: "references"},
		{RelPath: "scripts/run.sh", Kind: "scripts"},
	}}

	byKind := s.CompanionsByKind()
	if len(byKind["references"]) != 1 || byKind["references"][0] != "references/intro.md" {
		t.Errorf("CompanionsByKind[references] = %v", byKind["references"])
	}
	if len(byKind["scripts"]) != 1 {
		t.Errorf("CompanionsByKind[scripts] = %v", byKind["scripts"])
	}
	if byKind["templates"] != nil {
		t.Errorf("CompanionsByKind[templates] should be nil, got %v", byKind["templates"])
	}

	if !s.HasCompanionPath("references/intro.md") {
		t.Errorf("HasCompanionPath exact = false, want true")
	}
	if !s.HasCompanionPath("./references/intro.md") {
		t.Errorf("HasCompanionPath with ./ prefix = false, want true")
	}
	if s.HasCompanionPath("nope.md") {
		t.Errorf("HasCompanionPath missing = true, want false")
	}
}
