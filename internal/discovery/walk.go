package discovery

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	ignore "github.com/sabhiram/go-gitignore"

	"github.com/donaldgifford/claudelint/internal/artifact"
)

// Candidate is a discovered file plus its classified ArtifactKind.
// Parsing turns Candidates into typed Artifacts in phase 1.2.
type Candidate struct {
	// Path is repo-relative, slash-separated. It is the stable
	// identity reporters and diagnostics use.
	Path string

	// AbsPath is the absolute path on disk. Parsers read from this;
	// reporters ignore it.
	AbsPath string

	// Kind is the classification from Classify.
	Kind artifact.ArtifactKind
}

// Options configures the walker. Zero-value Options is a valid walker
// rooted at the current working directory with default ignore rules.
type Options struct {
	// Extra ignore globs from config (ignore.paths in .claudelint.hcl).
	// Interpreted with gitignore syntax.
	ExtraIgnore []string

	// GlobalIgnorePath overrides the path to the user-global gitignore
	// file (usually $XDG_CONFIG_HOME/git/ignore or ~/.config/git/ignore).
	// Empty string uses the default lookup.
	GlobalIgnorePath string
}

// Walker walks a repository root, honoring gitignore semantics, and
// returns Candidates for every recognized artifact. One Walker per
// run: it caches the global and excludes matchers.
type Walker struct {
	opts        Options
	globalMatch *ignore.GitIgnore
}

// New returns a Walker configured with opts. An empty Options is
// valid.
func New(opts Options) *Walker {
	return &Walker{opts: opts}
}

// Walk walks root and returns every Candidate it finds, sorted by Path
// for stable output.
func (w *Walker) Walk(root string) ([]Candidate, error) {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolve root: %w", err)
	}

	info, err := os.Stat(absRoot)
	if err != nil {
		return nil, fmt.Errorf("stat root: %w", err)
	}

	if !info.IsDir() {
		return walkFile(absRoot, root), nil
	}

	if err := w.ensureGlobal(); err != nil {
		return nil, err
	}

	stack := newMatcherStack(w.globalMatch)
	if exclude, err := loadExcludeFile(absRoot); err != nil {
		return nil, err
	} else if exclude != nil {
		stack.add(exclude)
	}
	if extra := w.extraMatcher(); extra != nil {
		stack.add(extra)
	}
	stack.pushDirIgnore("", absRoot)

	out, err := walkDir(absRoot, stack)
	if err != nil {
		return nil, err
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out, nil
}

// handleWalkErr decides whether a per-entry WalkDir error is fatal.
// Permission-denied on a single file or directory is skipped silently,
// matching `git status` behaviour on unreadable entries; any other
// error bubbles up.
func handleWalkErr(walkErr error) error {
	if errors.Is(walkErr, fs.ErrPermission) {
		return nil
	}
	return walkErr
}

// walkDir runs filepath.WalkDir with a callback that skips ignored
// paths, pushes nested .gitignore files onto the stack, and emits a
// Candidate for each classified file.
func walkDir(absRoot string, stack *matcherStack) ([]Candidate, error) {
	var out []Candidate
	err := filepath.WalkDir(absRoot, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return handleWalkErr(walkErr)
		}

		rel, err := filepath.Rel(absRoot, path)
		if err != nil {
			return fmt.Errorf("relativize %q: %w", path, err)
		}
		rel = filepath.ToSlash(rel)
		if rel == "." {
			return nil
		}

		if d.IsDir() {
			if d.Name() == ".git" {
				return fs.SkipDir
			}
			if stack.matchFile(rel) {
				return fs.SkipDir
			}
			stack.pushDirIgnore(rel, path)
			return nil
		}
		if stack.matchFile(rel) {
			return nil
		}
		if kind, ok := Classify(rel); ok {
			out = append(out, Candidate{Path: rel, AbsPath: path, Kind: kind})
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk %s: %w", absRoot, err)
	}
	return out, nil
}

// walkFile handles the case where the walker is handed a single file
// path rather than a directory. It bypasses gitignore — the user
// asked explicitly for this file.
func walkFile(absPath, userPath string) []Candidate {
	rel := filepath.ToSlash(filepath.Base(absPath))
	kind, ok := Classify(rel)
	if !ok {
		return nil
	}
	return []Candidate{{Path: userPath, AbsPath: absPath, Kind: kind}}
}

// ensureGlobal loads the user-global gitignore exactly once per walker.
func (w *Walker) ensureGlobal() error {
	if w.globalMatch != nil {
		return nil
	}
	p := w.opts.GlobalIgnorePath
	if p == "" {
		p = defaultGlobalIgnorePath()
	}
	if p == "" {
		return nil
	}
	if _, err := os.Stat(p); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("stat global ignore: %w", err)
	}
	m, err := ignore.CompileIgnoreFile(p)
	if err != nil {
		return fmt.Errorf("load global ignore %s: %w", p, err)
	}
	w.globalMatch = m
	return nil
}

func (w *Walker) extraMatcher() *ignore.GitIgnore {
	if len(w.opts.ExtraIgnore) == 0 {
		return nil
	}
	return ignore.CompileIgnoreLines(w.opts.ExtraIgnore...)
}

// defaultGlobalIgnorePath returns the path to the user-global gitignore
// following the same lookup git itself uses: $XDG_CONFIG_HOME/git/ignore,
// then $HOME/.config/git/ignore. Returns "" if neither is resolvable.
func defaultGlobalIgnorePath() string {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "git", "ignore")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".config", "git", "ignore")
}

// loadExcludeFile returns the matcher for .git/info/exclude. A
// missing file returns (nil, nil) — that path simply contributes
// nothing. A read or parse failure is a real error.
func loadExcludeFile(absRoot string) (*ignore.GitIgnore, error) {
	p := filepath.Join(absRoot, ".git", "info", "exclude")
	info, err := os.Stat(p)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil //nolint:nilnil // absence is not an error
		}
		return nil, fmt.Errorf("stat exclude: %w", err)
	}
	if info.IsDir() {
		return nil, nil //nolint:nilnil // unlikely but not an error
	}
	m, err := ignore.CompileIgnoreFile(p)
	if err != nil {
		return nil, fmt.Errorf("load exclude %s: %w", p, err)
	}
	return m, nil
}

// matcherStack layers gitignore matchers so nested .gitignore files
// are scoped to their subtree. Base matchers (global, exclude, extra)
// evaluate against repo-relative paths; nested entries evaluate
// against paths relative to the directory that carries them, matching
// git's semantics.
type matcherStack struct {
	base    []*ignore.GitIgnore
	nested  []nestedMatcher
	visited map[string]struct{}
}

type nestedMatcher struct {
	// dir is the slash-separated path of the directory that contains
	// the .gitignore, relative to the walk root.
	dir     string
	matcher *ignore.GitIgnore
}

func newMatcherStack(entries ...*ignore.GitIgnore) *matcherStack {
	base := make([]*ignore.GitIgnore, 0, len(entries))
	for _, m := range entries {
		if m != nil {
			base = append(base, m)
		}
	}
	return &matcherStack{base: base, visited: make(map[string]struct{})}
}

func (s *matcherStack) add(m *ignore.GitIgnore) {
	if m != nil {
		s.base = append(s.base, m)
	}
}

// pushDirIgnore loads dir/.gitignore if present and adds it to the
// stack. It is idempotent across the same dir so callers can call it
// on every directory encountered.
func (s *matcherStack) pushDirIgnore(rel, absDir string) {
	if _, seen := s.visited[rel]; seen {
		return
	}
	s.visited[rel] = struct{}{}

	p := filepath.Join(absDir, ".gitignore")
	info, err := os.Stat(p)
	if err != nil || info.IsDir() {
		return
	}
	m, err := ignore.CompileIgnoreFile(p)
	if err != nil {
		return
	}
	s.nested = append(s.nested, nestedMatcher{dir: rel, matcher: m})
}

func (s *matcherStack) matchFile(rel string) bool {
	for _, m := range s.base {
		if m.MatchesPath(rel) {
			return true
		}
	}
	for _, nm := range s.nested {
		sub, ok := relativeTo(nm.dir, rel)
		if !ok {
			continue
		}
		if nm.matcher.MatchesPath(sub) {
			return true
		}
	}
	return false
}

// relativeTo trims the dir prefix from rel. Returns ("", false) if
// rel does not live under dir.
func relativeTo(dir, rel string) (string, bool) {
	if dir == "" {
		return rel, true
	}
	if rel == dir {
		return "", false
	}
	if strings.HasPrefix(rel, dir+"/") {
		return rel[len(dir)+1:], true
	}
	return "", false
}
