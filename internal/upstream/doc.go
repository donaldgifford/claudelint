// Package upstream extracts a small, deterministic digest of the
// Claude Code and Agent Skills specifications from their published
// sources, diffs it against the digest committed in this repository,
// and reports the delta.
//
// claudelint encodes those specifications as hand-maintained Go
// constants and parser key lists in internal/artifact. Anthropic ships
// Claude Code several times a week and the public documentation is the
// only authoritative spec, so those constants go stale silently. This
// package is the upstream analogue of the ruleset fingerprint
// guardrail: regenerate, diff, and fail loudly with the identifier to
// update.
//
// The pipeline is fetch, extract, diff:
//
//	Fetch    retrieves every Source into a work directory.
//	Extract  runs one extractor per (source, digest section).
//	Diff     compares two digests structurally, not textually.
//
// Only the derived digest and a source lockfile are committed. Raw
// upstream pages are fetched fresh on every run and retained as a
// workflow artifact; they are never checked in.
//
// The package imports nothing else from this module. The guardrail
// test in Phase 2 depends on internal/artifact, not the other way
// round, so the specification data never becomes a build-time
// dependency of the linter itself.
//
// See DESIGN-0006 and IMPL-0005 under docs/ for the full design.
package upstream
