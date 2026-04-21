package artifact

import "testing"

// TestAllKindsStable guards the order and contents of AllKinds so any
// accidental reorder (which would churn user-visible output) fails
// loudly in a single-line diff.
func TestAllKindsStable(t *testing.T) {
	want := []ArtifactKind{
		KindClaudeMD,
		KindSkill,
		KindCommand,
		KindAgent,
		KindHook,
		KindPlugin,
	}
	got := AllKinds()
	if len(got) != len(want) {
		t.Fatalf("AllKinds() returned %d kinds, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("AllKinds()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

// TestAllKindsUnique catches a duplicate constant that would otherwise
// silently shadow another rule's AppliesTo.
func TestAllKindsUnique(t *testing.T) {
	seen := make(map[ArtifactKind]struct{}, len(AllKinds()))
	for _, k := range AllKinds() {
		if _, dup := seen[k]; dup {
			t.Errorf("duplicate kind %q", k)
		}
		seen[k] = struct{}{}
	}
}
