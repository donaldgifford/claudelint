package marketplace

import (
	"fmt"

	"github.com/donaldgifford/claudelint/internal/artifact"
	"github.com/donaldgifford/claudelint/internal/diag"
	"github.com/donaldgifford/claudelint/internal/rules"
)

func init() { rules.Register(&externalSourceSkipped{}) }

// externalSourceSkipped emits an info diagnostic for each plugins[]
// entry whose source content genuinely cannot be checked locally:
// remote string shorthands and the github, url, git-subdir, npm,
// archive, and command object kinds. claudelint validates the source's
// structure (see marketplace/plugin-source-valid) but does not fetch
// remote content — this rule surfaces the entry so users know it was
// noticed and skipped, not silently ignored. Local paths are checked in
// place; absent or invalid sources are plugin-source-valid findings,
// not skips, and are not double-reported here.
//
// The wording is kind-aware because the reasons differ. Most kinds are
// unlintable because the bytes live somewhere else. A command source is
// unlintable because the bytes do not exist yet: they are whatever the
// command prints when Claude Code runs it, which a linter will not do.
type externalSourceSkipped struct{}

func (*externalSourceSkipped) ID() string                     { return "marketplace/external-source-skipped" }
func (*externalSourceSkipped) Category() string               { return categorySchema }
func (*externalSourceSkipped) DefaultSeverity() diag.Severity { return diag.SeverityInfo }
func (*externalSourceSkipped) DefaultOptions() map[string]any { return nil }
func (*externalSourceSkipped) AppliesTo() []artifact.ArtifactKind {
	return []artifact.ArtifactKind{artifact.KindMarketplace}
}

func (*externalSourceSkipped) HelpURI() string {
	return rules.DefaultHelpURI("marketplace/external-source-skipped")
}

func (r *externalSourceSkipped) Check(_ rules.Context, a artifact.Artifact) []diag.Diagnostic {
	m, ok := a.(*artifact.Marketplace)
	if !ok {
		return nil
	}
	var out []diag.Diagnostic
	for i := range m.Plugins {
		p := &m.Plugins[i]
		note, skipped := skipNote(p)
		if !skipped {
			continue
		}
		rng := p.SourceRange
		if rng.IsZero() {
			rng = p.NameRange
		}
		out = append(out, diag.Diagnostic{
			RuleID:  r.ID(),
			Path:    m.Path(),
			Range:   rng,
			Message: fmt.Sprintf("plugins[%d] %s", i, note),
		})
	}
	return out
}

// skipNote renders the whole notice for one source, because the reason
// a source is unlintable is part of what the reader needs to hear.
// skipped is false for local paths and for absent or invalid sources.
// The legacy-string fallback covers hand-built artifacts that carry a
// remote shorthand in Source without a parsed SourceInfo.
func skipNote(p *artifact.MarketplacePlugin) (note string, skipped bool) {
	src := p.SourceInfo
	switch src.Kind {
	case artifact.SourceExternalString:
		return remoteNote(fmt.Sprintf("%q", p.Source)), true
	case artifact.SourceGitHub:
		return remoteNote(fmt.Sprintf("github %q", src.Repo)), true
	case artifact.SourceURL:
		return remoteNote(fmt.Sprintf("url %q", src.URL)), true
	case artifact.SourceGitSubdir:
		return remoteNote(fmt.Sprintf("git-subdir %q path %q", src.URL, src.Path)), true
	case artifact.SourceNPM:
		return remoteNote(fmt.Sprintf("npm %q", src.Package)), true
	case artifact.SourceArchive:
		return fmt.Sprintf(
			"source archive %q is remote — the archive is not downloaded, unpacked, or linted by claudelint",
			src.URL), true
	case artifact.SourceCommand:
		return fmt.Sprintf(
			"source command %q is generated — claudelint does not run it, so the plugin it produces is not linted",
			src.Command), true
	case artifact.SourceAbsent:
		if p.Source != "" && p.Resolved == "" {
			return remoteNote(fmt.Sprintf("%q", p.Source)), true
		}

		return "", false
	default:
		return "", false
	}
}

// remoteNote is the wording shared by every kind whose content lives on
// another machine.
func remoteNote(locator string) string {
	return "source " + locator + " is remote — content not fetched or linted by claudelint"
}
