package artifact

import "testing"

func TestNewBaseLineIndex(t *testing.T) {
	src := []byte("line1\nline2\nline3")
	b := NewBase("x", src)

	tests := []struct {
		name              string
		offset            int
		wantLine, wantCol int
	}{
		{"start of line 1", 0, 1, 1},
		{"end of line 1", 5, 1, 6},
		{"start of line 2", 6, 2, 1},
		{"middle of line 2", 8, 2, 3},
		{"start of line 3", 12, 3, 1},
		{"end of source", len(src), 3, 6},
		{"negative clamps to start", -3, 1, 1},
		{"over-long clamps to end", 10000, 3, 6},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := b.ResolvePosition(tt.offset)
			if p.Line != tt.wantLine || p.Column != tt.wantCol {
				t.Errorf("ResolvePosition(%d) = line %d col %d, want line %d col %d",
					tt.offset, p.Line, p.Column, tt.wantLine, tt.wantCol)
			}
		})
	}
}

func TestNewBaseSourceIsCopied(t *testing.T) {
	src := []byte("hello")
	b := NewBase("x", src)
	src[0] = 'H'
	if got := string(b.Source()); got != "hello" {
		t.Errorf("Source() = %q, want unchanged copy %q", got, "hello")
	}
}

func TestResolveRange(t *testing.T) {
	b := NewBase("x", []byte("abcdef\nghi"))
	r := b.ResolveRange(4, 8) // chars "ef\ng"
	if r.Start.Line != 1 || r.Start.Column != 5 {
		t.Errorf("start = %+v, want line 1 col 5", r.Start)
	}
	if r.End.Line != 2 || r.End.Column != 2 {
		t.Errorf("end = %+v, want line 2 col 2", r.End)
	}
}
