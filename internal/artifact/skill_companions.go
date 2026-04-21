package artifact

import (
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// IndexSkillCompanions scans the skill directory for companion files
// (anything below references/, scripts/, or templates/) and populates
// s.Companions with one entry per file.
//
// skillDir is the absolute path to the directory that holds SKILL.md —
// e.g. /repo/.claude/skills/writer. Files outside the three canonical
// subdirectories are ignored; companion scanning is not a lint pass
// and should not surface unrelated files.
//
// Errors reading individual files are swallowed: a skill with a
// broken symlink under references/ is still a valid skill for the
// purposes of linting SKILL.md itself.
func IndexSkillCompanions(s *Skill, skillDir string) error {
	kinds := []string{"references", "scripts", "templates"}

	var companions []Companion
	for _, k := range kinds {
		root := filepath.Join(skillDir, k)
		if _, err := fs.Stat(dirFS(skillDir), k); err != nil {
			continue
		}
		err := filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return nil //nolint:nilerr // a missing sub-entry is not fatal
			}
			if d.IsDir() {
				return nil
			}
			rel, err := filepath.Rel(skillDir, path)
			if err != nil {
				return nil //nolint:nilerr // unreachable in practice
			}
			companions = append(companions, Companion{
				RelPath: filepath.ToSlash(rel),
				Kind:    k,
			})
			return nil
		})
		if err != nil {
			return err
		}
	}

	sort.Slice(companions, func(i, j int) bool {
		return companions[i].RelPath < companions[j].RelPath
	})
	s.Companions = companions
	return nil
}

// dirFS is a small wrapper so IndexSkillCompanions can be tested with
// a filesystem fixture if needed. It delegates to os.DirFS.
func dirFS(dir string) fs.FS { return os.DirFS(dir) }

// CompanionsByKind returns a convenience view grouping Companions by
// their Kind. The returned map is always non-nil; missing kinds map
// to a nil slice.
func (s *Skill) CompanionsByKind() map[string][]string {
	out := make(map[string][]string, 3)
	for _, c := range s.Companions {
		out[c.Kind] = append(out[c.Kind], c.RelPath)
	}
	return out
}

// HasCompanionPath reports whether rel (slash-separated, relative to
// the skill directory) is an indexed companion. Rules use it to
// validate references inside SKILL.md without re-walking disk.
func (s *Skill) HasCompanionPath(rel string) bool {
	rel = strings.TrimPrefix(rel, "./")
	for _, c := range s.Companions {
		if c.RelPath == rel {
			return true
		}
	}
	return false
}
