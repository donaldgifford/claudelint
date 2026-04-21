package artifact

import (
	"testing"

	"github.com/donaldgifford/claudelint/internal/diag"
)

func TestConcreteKinds(t *testing.T) {
	tests := []struct {
		name string
		a    Artifact
		want ArtifactKind
	}{
		{"ClaudeMD", &ClaudeMD{}, KindClaudeMD},
		{"Skill", &Skill{}, KindSkill},
		{"Command", &Command{}, KindCommand},
		{"Agent", &Agent{}, KindAgent},
		{"Hook", &Hook{}, KindHook},
		{"Plugin", &Plugin{}, KindPlugin},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.a.Kind(); got != tt.want {
				t.Errorf("%T.Kind() = %q, want %q", tt.a, got, tt.want)
			}
		})
	}
}

func TestFrontmatterKeyRange(t *testing.T) {
	f := Frontmatter{}
	if got := f.KeyRange("missing"); !got.IsZero() {
		t.Errorf("KeyRange on nil Keys = %+v, want zero", got)
	}

	nameRange := diag.Range{Start: diag.Position{Line: 2, Column: 1, Offset: 5}}
	f.Keys = map[string]diag.Range{"name": nameRange}
	if got := f.KeyRange("name"); got != nameRange {
		t.Errorf("KeyRange(\"name\") = %+v, want %+v", got, nameRange)
	}
	if got := f.KeyRange("description"); !got.IsZero() {
		t.Errorf("KeyRange(\"description\") = %+v, want zero", got)
	}
}
