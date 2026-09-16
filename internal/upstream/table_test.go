package upstream_test

import (
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/donaldgifford/claudelint/internal/upstream"
)

func TestFirstTable(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		src        string
		wantOK     bool
		wantHeader []string
		wantRows   [][]string
	}{
		{
			name: "plain table",
			src: "| Field | Required |\n" +
				"| --- | --- |\n" +
				"| `name` | Yes |\n" +
				"| `description` | Recommended |\n",
			wantOK:     true,
			wantHeader: []string{"field", "required"},
			wantRows:   [][]string{{"`name`", "Yes"}, {"`description`", "Recommended"}},
		},
		{
			name: "left-aligned delimiter row",
			src: "| Tool | Description |\n" +
				"| :--------- | :------------------- |\n" +
				"| `Bash` | Executes shell commands |\n",
			wantOK:     true,
			wantHeader: []string{"tool", "description"},
			wantRows:   [][]string{{"`Bash`", "Executes shell commands"}},
		},
		{
			name: "every alignment marker",
			src: "| a | b | c |\n" +
				"| :--- | :---: | ---: |\n" +
				"| 1 | 2 | 3 |\n",
			wantOK:     true,
			wantHeader: []string{"a", "b", "c"},
			wantRows:   [][]string{{"1", "2", "3"}},
		},
		{
			name: "trailing whitespace on every line",
			src: "| Field | Required |   \n" +
				"| --- | --- |\t\n" +
				"| `name` | Yes |  \n",
			wantOK:     true,
			wantHeader: []string{"field", "required"},
			wantRows:   [][]string{{"`name`", "Yes"}},
		},
		{
			name: "escaped pipe inside a cell",
			src: "| Field | Type |\n" +
				"| --- | --- |\n" +
				`| ` + "`hooks`" + ` | string\|array\|object |` + "\n",
			wantOK:     true,
			wantHeader: []string{"field", "type"},
			wantRows:   [][]string{{"`hooks`", "string|array|object"}},
		},
		{
			name: "nested backticks in a description cell",
			src: "| Field | Description |\n" +
				"| --- | --- |\n" +
				"| `shell` | Shell to use for `` !`command` `` blocks |\n",
			wantOK:     true,
			wantHeader: []string{"field", "description"},
			wantRows:   [][]string{{"`shell`", "Shell to use for `` !`command` `` blocks"}},
		},
		{
			name: "no leading or trailing delimiters",
			src: "Field | Required\n" +
				"--- | ---\n" +
				"`name` | Yes\n",
			wantOK:     true,
			wantHeader: []string{"field", "required"},
			wantRows:   [][]string{{"`name`", "Yes"}},
		},
		{
			name: "short row is padded",
			src: "| a | b | c |\n" +
				"| --- | --- | --- |\n" +
				"| 1 |\n",
			wantOK:     true,
			wantHeader: []string{"a", "b", "c"},
			wantRows:   [][]string{{"1", "", ""}},
		},
		{
			name:   "pipes without a delimiter row are not a table",
			src:    "| this | is | prose |\nand so is this\n",
			wantOK: false,
		},
		{
			name:   "delimiter row with the wrong cell count",
			src:    "| a | b |\n| --- |\n| 1 | 2 |\n",
			wantOK: false,
		},
		{
			name:   "no table at all",
			src:    "Just a paragraph.\n\nAnd another.\n",
			wantOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, ok := upstream.FirstTable(tt.src)
			if ok != tt.wantOK {
				t.Fatalf("FirstTable ok = %v, want %v", ok, tt.wantOK)
			}
			if !tt.wantOK {
				return
			}
			if !slices.Equal(got.Header, tt.wantHeader) {
				t.Errorf("Header = %q, want %q", got.Header, tt.wantHeader)
			}
			if len(got.Rows) != len(tt.wantRows) {
				t.Fatalf("got %d rows, want %d: %q", len(got.Rows), len(tt.wantRows), got.Rows)
			}
			for i := range tt.wantRows {
				if !slices.Equal(got.Rows[i], tt.wantRows[i]) {
					t.Errorf("row %d = %q, want %q", i, got.Rows[i], tt.wantRows[i])
				}
			}
		})
	}
}

func TestTableEndsAtHeadingOrBlankLine(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		src      string
		wantRows int
		wantMore int
	}{
		{
			name: "ends at a heading",
			src: "| a |\n| --- |\n| 1 |\n" +
				"## Next section\n" +
				"| b |\n| --- |\n| 2 |\n",
			wantRows: 1,
			wantMore: 2,
		},
		{
			name: "ends at a blank line",
			src: "| a |\n| --- |\n| 1 |\n" +
				"\n" +
				"| b |\n| --- |\n| 2 |\n",
			wantRows: 1,
			wantMore: 2,
		},
		{
			name:     "ends at prose without a pipe",
			src:      "| a |\n| --- |\n| 1 |\nSome prose.\n| b |\n| --- |\n| 2 |\n",
			wantRows: 1,
			wantMore: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tables := upstream.Tables(tt.src)
			if len(tables) != tt.wantMore {
				t.Fatalf("got %d tables, want %d", len(tables), tt.wantMore)
			}
			if len(tables[0].Rows) != tt.wantRows {
				t.Errorf("first table has %d rows, want %d", len(tables[0].Rows), tt.wantRows)
			}
		})
	}
}

func TestTablesIgnoreFencedBlocks(t *testing.T) {
	t.Parallel()

	src := "```markdown\n" +
		"| fake | table |\n| --- | --- |\n| not | extracted |\n" +
		"```\n" +
		"| real | table |\n| --- | --- |\n| yes | extracted |\n"

	tables := upstream.Tables(src)
	if len(tables) != 1 {
		t.Fatalf("got %d tables, want 1 (the fenced one must be skipped)", len(tables))
	}
	if got := tables[0].Cell(0, 0); got != "yes" {
		t.Errorf("first cell = %q, want %q", got, "yes")
	}
}

func TestTableColumnAndHeaders(t *testing.T) {
	t.Parallel()

	src := "| Field | Required | Description |\n| --- | --- | --- |\n| `name` | Yes | The name |\n"

	tbl, ok := upstream.FirstTable(src)
	if !ok {
		t.Fatal("FirstTable found nothing")
	}

	if got := tbl.Column("Required"); got != 1 {
		t.Errorf("Column(Required) = %d, want 1", got)
	}
	if got := tbl.Column("  REQUIRED "); got != 1 {
		t.Errorf("Column is not case- and space-insensitive: got %d", got)
	}
	if got := tbl.Column("missing"); got != -1 {
		t.Errorf("Column(missing) = %d, want -1", got)
	}
	if !tbl.HasHeaders("field", "required") {
		t.Error("HasHeaders(field, required) = false")
	}
	if tbl.HasHeaders("field", "type") {
		t.Error("HasHeaders accepted a column that is not there")
	}
	if tbl.HasHeaders("field", "required", "description", "extra") {
		t.Error("HasHeaders accepted more columns than the table has")
	}
	if got := tbl.Cell(0, 2); got != "The name" {
		t.Errorf("Cell(0,2) = %q", got)
	}
	if got := tbl.Cell(9, 0); got != "" {
		t.Errorf("Cell out of range = %q, want empty", got)
	}
	if got := tbl.Cell(0, 9); got != "" {
		t.Errorf("Cell out of range = %q, want empty", got)
	}
}

func TestSection(t *testing.T) {
	t.Parallel()

	const page = `# Page title

Intro prose.

## Hook events

### PreToolUse

Runs before a tool.

### PostToolUse

Runs after a tool.

## Next top-level section

### Not a hook event
`

	tests := []struct {
		name        string
		anchor      string
		wantOK      bool
		wantContain []string
		wantMissing []string
	}{
		{
			name:        "stops at the next equal-level heading",
			anchor:      `^## Hook events\s*$`,
			wantOK:      true,
			wantContain: []string{"## Hook events", "### PreToolUse", "### PostToolUse"},
			wantMissing: []string{"## Next top-level section", "Not a hook event"},
		},
		{
			name:        "a deeper anchor stops at the next sibling",
			anchor:      `^### PreToolUse\s*$`,
			wantOK:      true,
			wantContain: []string{"Runs before a tool."},
			wantMissing: []string{"PostToolUse"},
		},
		{
			name:        "last section runs to the end",
			anchor:      `^## Next top-level section\s*$`,
			wantOK:      true,
			wantContain: []string{"Not a hook event"},
		},
		{
			name:   "missing anchor",
			anchor: `^## Nonexistent\s*$`,
			wantOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, ok := upstream.Section([]byte(page), regexp.MustCompile(tt.anchor))
			if ok != tt.wantOK {
				t.Fatalf("Section ok = %v, want %v", ok, tt.wantOK)
			}
			if !tt.wantOK {
				return
			}
			for _, want := range tt.wantContain {
				if !strings.Contains(got, want) {
					t.Errorf("section is missing %q:\n%s", want, got)
				}
			}
			for _, bad := range tt.wantMissing {
				if strings.Contains(got, bad) {
					t.Errorf("section leaked past its boundary, found %q:\n%s", bad, got)
				}
			}
		})
	}
}

// TestSectionIgnoresHeadingsInsideFences pins the reason this scanner
// tracks fence state: the upstream pages embed shell snippets whose
// comment lines are indistinguishable from ATX headings.
func TestSectionIgnoresHeadingsInsideFences(t *testing.T) {
	t.Parallel()

	const page = "## Restrict skill access\n\n" +
		"```bash\n" +
		"# Add to deny rules:\n" +
		"claude config add deny Skill\n" +
		"## Not a heading either\n" +
		"```\n\n" +
		"Still inside the section.\n\n" +
		"## Real next section\n"

	got, ok := upstream.Section([]byte(page), regexp.MustCompile(`^## Restrict skill access\s*$`))
	if !ok {
		t.Fatal("Section did not find the anchor")
	}
	if !strings.Contains(got, "Still inside the section.") {
		t.Errorf("section ended early at a fenced comment:\n%s", got)
	}
	if strings.Contains(got, "Real next section") {
		t.Errorf("section ran past its real boundary:\n%s", got)
	}
}

func TestSectionNormalisesCRLF(t *testing.T) {
	t.Parallel()

	page := []byte("## Anchor\r\nbody line\r\n## Next\r\n")

	got, ok := upstream.Section(page, regexp.MustCompile(`^## Anchor\s*$`))
	if !ok {
		t.Fatal("Section did not find the anchor in a CRLF document")
	}
	if strings.ContainsRune(got, '\r') {
		t.Errorf("section kept a carriage return: %q", got)
	}
	if got != "## Anchor\nbody line\n" {
		t.Errorf("section = %q", got)
	}
}

func TestHeadings(t *testing.T) {
	t.Parallel()

	const src = "## Hook events\n\n" +
		"### PreToolUse\n\n" +
		"#### A sub-heading\n\n" +
		"### PostToolUse\n\n" +
		"```bash\n### Not a heading\n```\n"

	got := upstream.Headings(src, 3)
	want := []string{"PreToolUse", "PostToolUse"}

	if !slices.Equal(got, want) {
		t.Errorf("Headings(level 3) = %q, want %q", got, want)
	}
	if got := upstream.Headings(src, 2); !slices.Equal(got, []string{"Hook events"}) {
		t.Errorf("Headings(level 2) = %q", got)
	}
	if got := upstream.Headings(src, 5); len(got) != 0 {
		t.Errorf("Headings(level 5) = %q, want none", got)
	}
}

func TestName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		in, want string
	}{
		{in: "`name`", want: "name"},
		{in: "  `allowed-tools`  ", want: "allowed-tools"},
		{in: "``name``", want: "name"},
		{in: "name", want: "name"},
		{in: "", want: ""},
		{in: "`", want: "`"},
		{in: "`PreToolUse` (deprecated)", want: "`PreToolUse` (deprecated)"},
	}

	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			t.Parallel()

			if got := upstream.Name(tt.in); got != tt.want {
				t.Errorf("Name(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestBacktickedTokens(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   string
		want []string
	}{
		{
			name: "enum values in a description cell",
			in:   "[Model](#choose-a-model) to use: `sonnet`, `opus`, `haiku`, `fable`, or `inherit`",
			want: []string{"sonnet", "opus", "haiku", "fable", "inherit"},
		},
		{
			name: "no tokens",
			in:   "plain prose",
			want: nil,
		},
		{
			name: "duplicates are preserved in order",
			in:   "`a` then `b` then `a`",
			want: []string{"a", "b", "a"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := upstream.BacktickedTokens(tt.in)
			if len(got) == 0 && len(tt.want) == 0 {
				return
			}
			if !slices.Equal(got, tt.want) {
				t.Errorf("BacktickedTokens(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}
