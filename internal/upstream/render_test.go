package upstream_test

import (
	"bytes"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/donaldgifford/claudelint/internal/upstream"
)

// renderEmbedded is the render every test here starts from: the
// committed digest, the committed acknowledgements, and the shipped
// coverage table.
func renderEmbedded(t *testing.T) []byte {
	t.Helper()

	digest, ack, err := upstream.LoadEmbedded()
	if err != nil {
		t.Fatalf("LoadEmbedded() error = %v", err)
	}

	page, err := upstream.RenderSpec(digest, ack, upstream.DefaultCoverage())
	if err != nil {
		t.Fatalf("RenderSpec() error = %v", err)
	}

	return page
}

// TestRenderedPageMatchesTheCommittedFile is the staleness guard that
// runs on every `just test`, not just when someone remembers to run
// `render --check`. A digest change that nobody re-rendered fails here.
func TestRenderedPageMatchesTheCommittedFile(t *testing.T) {
	t.Parallel()

	committed, err := os.ReadFile(filepath.Join("..", "..", upstream.SpecPagePath))
	if err != nil {
		t.Fatalf("read the committed page: %v", err)
	}

	if !bytes.Equal(renderEmbedded(t), committed) {
		t.Errorf("%s is out of date; regenerate it with `just spec-render`", upstream.SpecPagePath)
	}
}

// TestRenderIsDeterministic is what lets --check mean anything. Every
// map in the digest has to be walked in sorted order, and there is no
// timestamp anywhere on the page.
func TestRenderIsDeterministic(t *testing.T) {
	t.Parallel()

	first := renderEmbedded(t)
	for range 5 {
		if !bytes.Equal(renderEmbedded(t), first) {
			t.Fatal("two renders of the same digest differ")
		}
	}
}

func TestRenderedPageShape(t *testing.T) {
	t.Parallel()

	page := string(renderEmbedded(t))

	t.Run("starts with the frontmatter Starlight requires", func(t *testing.T) {
		t.Parallel()

		if !strings.HasPrefix(page, "---\ntitle: Upstream spec\n---\n") {
			t.Errorf("page does not start with title frontmatter:\n%.80s", page)
		}
	})

	t.Run("says it is generated", func(t *testing.T) {
		t.Parallel()

		if !strings.Contains(page, "Do not edit.") {
			t.Error("page does not warn that it is generated")
		}
	})

	t.Run("links the deprecated tools table rather than duplicating it", func(t *testing.T) {
		t.Parallel()

		if !strings.Contains(page, "rules.md#deprecated-and-removed-tools") {
			t.Error("page does not link the deprecated tools table")
		}
		if strings.Contains(page, "was removed in v") {
			t.Error("page duplicates the deprecated tools table instead of linking it")
		}
	})

	t.Run("carries the verified-against line", func(t *testing.T) {
		t.Parallel()

		if !strings.Contains(page, "Verified against Claude Code v") {
			t.Error("page does not say which runtime version it was verified against")
		}
	})

	t.Run("has one section per artifact kind", func(t *testing.T) {
		t.Parallel()

		for _, heading := range []string{
			"## Tools", "## Hooks", "## Skills and commands", "## Agents",
			"## Plugins", "## Marketplaces", "## MCP servers", "## Portable skills",
		} {
			if !strings.Contains(page, heading+"\n") {
				t.Errorf("page has no %q section", heading)
			}
		}
	})
}

// TestRenderReportsCoverageInThreeStates is the point of the page. A
// documented field nothing reads has to be visibly different from one
// claudelint has no opinion about.
func TestRenderReportsCoverageInThreeStates(t *testing.T) {
	t.Parallel()

	page := string(renderEmbedded(t))

	// tools.builtin is fully covered, so every row reads yes.
	tools := column(between(page, "## Tools", "## Hooks"), "claudelint")
	if len(tools) == 0 {
		t.Fatal("the tools table rendered no rows")
	}
	for _, got := range tools {
		if got != "yes" {
			t.Errorf("a documented tool reports %q; want yes", got)
		}
	}

	// skills.frontmatter carries documented fields no rule reads, each
	// with the reason that says which rule would.
	if !strings.Contains(page, "no rule consumes it") {
		t.Error("no acknowledgement reason reached the page")
	}

	// marketplace.fields has no coverage entry, so every row is the
	// no-opinion dash rather than a "no" the reader would act on.
	for _, got := range column(between(page, "## Marketplaces", "### Owner"), "claudelint") {
		if got != "—" {
			t.Errorf("an uncovered section reports %q; want the no-opinion dash", got)
		}
	}
}

// column returns the values under one header in the first table of a
// section, so an assertion names the column rather than counting pipes.
func column(section, header string) []string {
	var (
		idx = -1
		out []string
	)

	for _, line := range strings.Split(section, "\n") {
		if !strings.HasPrefix(line, "|") {
			continue
		}

		cells := splitRow(line)
		switch {
		case idx < 0:
			idx = slices.Index(cells, header)
		case strings.HasPrefix(cells[0], "---"):
			// The separator row carries no value.
		case idx < len(cells):
			out = append(out, cells[idx])
		}
	}

	return out
}

// splitRow splits one padded pipe row into trimmed cells.
func splitRow(line string) []string {
	cells := strings.Split(strings.Trim(line, "|"), "|")
	for i, c := range cells {
		cells[i] = strings.TrimSpace(c)
	}

	return cells
}

// TestRenderRejectsACoverageTypo keeps the table honest: a path that is
// not in the digest would silently render as "no opinion" forever.
func TestRenderRejectsACoverageTypo(t *testing.T) {
	t.Parallel()

	digest, ack, err := upstream.LoadEmbedded()
	if err != nil {
		t.Fatalf("LoadEmbedded() error = %v", err)
	}

	cov := upstream.DefaultCoverage()
	cov["tools.imaginary"] = map[string]struct{}{"Read": {}}

	if _, err := upstream.RenderSpec(digest, ack, cov); err == nil {
		t.Fatal("RenderSpec() accepted a coverage path that is not in the digest")
	} else if !strings.Contains(err.Error(), "tools.imaginary") {
		t.Errorf("error does not name the bad path: %v", err)
	}
}

// between returns the text between two markers, for asserting about one
// section without pinning the whole page.
func between(s, start, end string) string {
	i := strings.Index(s, start)
	if i < 0 {
		return ""
	}
	rest := s[i:]
	if j := strings.Index(rest, end); j >= 0 {
		return rest[:j]
	}

	return rest
}
