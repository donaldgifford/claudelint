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
// remote string shorthands and the github/url/git-subdir/npm object
// kinds. claudelint validates the source's structure (see
// marketplace/plugin-source-valid) but does not fetch remote content —
// this rule surfaces the entry so users know it was noticed and
// skipped, not silently ignored. Local paths are checked in place;
// absent or invalid sources are plugin-source-valid findings, not
// skips, and are not double-reported here.
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
		locator, remote := remoteLocator(p)
		if !remote {
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
			Message: fmt.Sprintf("plugins[%d] source %s is remote — content not fetched or linted by claudelint", i, locator),
		})
	}
	return out
}

// remoteLocator renders a remote source for the skip notice. remote
// is false for local paths and for absent/invalid sources. The
// legacy-string fallback covers hand-built artifacts that carry a
// remote shorthand in Source without a parsed SourceInfo.
func remoteLocator(p *artifact.MarketplacePlugin) (locator string, remote bool) {
	src := p.SourceInfo
	switch src.Kind {
	case artifact.SourceExternalString:
		return fmt.Sprintf("%q", p.Source), true
	case artifact.SourceGitHub:
		return fmt.Sprintf("github %q", src.Repo), true
	case artifact.SourceURL:
		return fmt.Sprintf("url %q", src.URL), true
	case artifact.SourceGitSubdir:
		return fmt.Sprintf("git-subdir %q path %q", src.URL, src.Path), true
	case artifact.SourceNPM:
		return fmt.Sprintf("npm %q", src.Package), true
	case artifact.SourceAbsent:
		if p.Source != "" && p.Resolved == "" {
			return fmt.Sprintf("%q", p.Source), true
		}
		return "", false
	default:
		return "", false
	}
}
