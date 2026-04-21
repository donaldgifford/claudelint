package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/hashicorp/hcl/v2/hclparse"
)

// Filename is the on-disk name discovery walks up the tree for.
const Filename = ".claudelint.hcl"

// LoadResult carries the loaded File plus the absolute path it came
// from. The path is exported so "explain" output and diagnostics can
// tell users which .claudelint.hcl was used.
type LoadResult struct {
	File *File
	Path string
}

// Load reads and decodes a .claudelint.hcl from explicitPath. When
// explicitPath is empty, FindConfig walks up from startDir until it
// finds a .claudelint.hcl (or runs out of parents). A missing config
// is not an error — the returned *LoadResult is nil and callers fall
// back to built-in defaults.
func Load(explicitPath, startDir string) (*LoadResult, error) {
	path := explicitPath
	if path == "" {
		found, err := FindConfig(startDir)
		if err != nil {
			return nil, err
		}
		if found == "" {
			return nil, nil //nolint:nilnil // absence is intentional
		}
		path = found
	}

	src, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config %s: %w", path, err)
	}
	file, diags := parseFile(path, src)
	if diags.HasErrors() {
		return nil, diagsError(diags)
	}
	if err := validate(file); err != nil {
		return nil, err
	}
	return &LoadResult{File: file, Path: path}, nil
}

// FindConfig walks up from startDir looking for a .claudelint.hcl.
// Returns the absolute path if found, an empty string if not. A
// failure to stat a directory is surfaced as an error.
func FindConfig(startDir string) (string, error) {
	dir, err := filepath.Abs(startDir)
	if err != nil {
		return "", fmt.Errorf("resolve start dir: %w", err)
	}
	for {
		candidate := filepath.Join(dir, Filename)
		info, err := os.Stat(candidate)
		switch {
		case err == nil && !info.IsDir():
			return candidate, nil
		case err != nil && !os.IsNotExist(err):
			return "", fmt.Errorf("stat %s: %w", candidate, err)
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", nil
		}
		dir = parent
	}
}

func parseFile(path string, src []byte) (*File, hcl.Diagnostics) {
	parser := hclparse.NewParser()
	f, diags := parser.ParseHCL(src, path)
	if diags.HasErrors() {
		return nil, diags
	}
	var out File
	diags = append(diags, gohcl.DecodeBody(f.Body, nil, &out)...)
	if diags.HasErrors() {
		return nil, diags
	}
	return &out, diags
}

// diagsError renders an hcl.Diagnostics into a single error that
// preserves each individual error's file/line/column. Callers print
// it as-is; the engine's schema/parse synthesis never sees it because
// config errors abort the run before any artifact is parsed.
func diagsError(diags hcl.Diagnostics) error {
	return fmt.Errorf("%s", diags.Error())
}
