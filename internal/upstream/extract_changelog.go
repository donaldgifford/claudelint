package upstream

import (
	"fmt"
	"regexp"
)

// changelogRelease matches the heading of one Claude Code release. The
// first match in the file is the newest.
var changelogRelease = regexp.MustCompile(`(?m)^##\s+(\d+\.\d+\.\d+)\s*$`)

// changelogLatest records the newest published Claude Code release.
//
// This is metadata: it leads the drift report so a reader sees how far
// the digest has moved, and it never counts toward the exit code.
type changelogLatest struct{}

var _ Extractor = changelogLatest{}

func (changelogLatest) Section() string        { return "meta.claude_code_latest" }
func (changelogLatest) Source() string         { return SrcClaudeCodeChangelog }
func (changelogLatest) Anchor() *regexp.Regexp { return nil }

func (changelogLatest) Extract(section string, out *Digest) error {
	m := changelogRelease.FindStringSubmatch(section)
	if m == nil {
		return fmt.Errorf("%w: no release heading in the changelog", ErrEmptySection)
	}

	out.Meta.ClaudeCodeLatest = m[1]

	return nil
}
